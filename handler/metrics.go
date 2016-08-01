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
    </div>
    <script type="text/javascript" src="//cdnjs.cloudflare.com/ajax/libs/jquery/3.1.0/jquery.slim.min.js"></script>
    <script type="text/javascript" src="//cdnjs.cloudflare.com/ajax/libs/d3/3.5.17/d3.min.js"></script>
    <script type="text/javascript" src="//cdnjs.cloudflare.com/ajax/libs/crossfilter/1.3.12/crossfilter.min.js"></script>
    <script type="text/javascript" src="//cdnjs.cloudflare.com/ajax/libs/dc/1.7.5/dc.min.js"></script>
    <script type="text/javascript">
        'use strict';
        var chartApps = dc.barChart("#roundTrips");
        var jsonURL = "/metrics/data";
        console.log(jsonURL);
        d3.json(jsonURL,
         function(error, metrics) {
          var latencies = metrics.round_trip
          var num_bins = 25;
          var range_max = d3.max(latencies);
          var bin_width = range_max / num_bins;

          var cf = crossfilter(latencies),
            latDim = cf.dimension(function(d) {return d;}),
            latGrouped = latDim.group(function(lat) { return Math.floor(lat/bin_width)*bin_width; }).reduceCount();

          chartApps
            .width(1000)
            .height(500)
            .x(d3.scale.linear().domain([0,range_max]))
            .brushOn(false)
            .xUnits(function() { return num_bins; })
            .yAxisLabel("Frequency")
            .xAxisLabel("Latency (s)")
            .dimension(latDim)
            .group(latGrouped);
            chartApps.render();
        });
    </script>
  </body>
<html>
`
