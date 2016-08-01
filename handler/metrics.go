package handler

import (
	"encoding/json"
	"net/http"

	"code.cloudfoundry.org/lager"
)

type MetricsData struct {
	Logger         lager.Logger
	SnapshotGetter func() interface{}
}

func (h *MetricsData) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.Logger.Session("handle-metrics")
	defer logger.Debug("done")

	snapshot := h.SnapshotGetter()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

type MetricsDisplay struct {
	Logger lager.Logger
}

func (h *MetricsDisplay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.Logger.Session("handle-metrics-display")
	defer logger.Debug("done")
	w.Write([]byte(body))
}

const body = `
<!DOCTYPE html>
<html lang="en">
  <head>
      <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
      <title>Metrics</title>
      <meta charset="UTF-8">
      <link rel="stylesheet" type="text/css" href="//cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.3.6/css/bootstrap.min.css">
      <link rel="stylesheet" type="text/css" href="//cdnjs.cloudflare.com/ajax/libs/dc/1.7.5/dc.css">
  </head>
  <body>
    <div class="container">
      <div id="roundTrips"></div>
      <div id="bandwidth"></div>
    </div>
    <script type="text/javascript" src="//cdnjs.cloudflare.com/ajax/libs/jquery/3.1.0/jquery.slim.min.js"></script>
    <script type="text/javascript" src="//cdnjs.cloudflare.com/ajax/libs/d3/3.5.17/d3.min.js"></script>
    <script type="text/javascript" src="//cdnjs.cloudflare.com/ajax/libs/crossfilter/1.3.12/crossfilter.min.js"></script>
    <script type="text/javascript" src="//cdnjs.cloudflare.com/ajax/libs/dc/1.7.5/dc.min.js"></script>
    <script type="text/javascript">
        'use strict';
        var chartRoundTrips = dc.barChart("#roundTrips");
        var chartBandwidth = dc.barChart("#bandwidth");
        var jsonURL = "/metrics/data";
        d3.json(jsonURL,
         function(error, metrics) {

          var latencies = metrics.round_trip;
          var cf = crossfilter(latencies),
            latDim = cf.dimension(function(d) {return d;}),
            latGrouped = latDim.group(function(lat) {
              return Math.floor(Math.log10(lat + 0.00001));
            }).reduceCount();
          var y_max = latGrouped.top(1)[0].value;
          chartRoundTrips
            .width(1000)
            .height(500)
            .x(d3.scale.linear().domain([-4,2]))
            .y(d3.scale.sqrt().domain([0,y_max+1]))
            .brushOn(false)
            .yAxisLabel("Frequency")
            .xAxisLabel("Latency (log seconds)")
            .dimension(latDim)
            .group(latGrouped);
            chartRoundTrips.render();

          var bandwidth = metrics.bandwidth;
          var cf = crossfilter(bandwidth),
            bwDim = cf.dimension(function(d) {return d/1000000;}),
            bwGrouped = bwDim.group(function(bw) {
              return Math.floor(bw);
            }).reduceCount();
          var y_max = bwGrouped.top(1)[0].value;
          chartBandwidth
            .width(1000)
            .height(500)
            .x(d3.scale.linear().domain([0,15]))
            .y(d3.scale.linear().domain([0,y_max+1]))
            .brushOn(false)
            .yAxisLabel("Frequency")
            .xAxisLabel("Bandwidth (MB/s)")
            .dimension(bwDim)
            .group(bwGrouped);
            chartBandwidth.render();
        });
    </script>
  </body>
<html>
`
