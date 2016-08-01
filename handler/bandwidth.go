package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/rosenhouse/reflex/science"

	"code.cloudfoundry.org/lager"
)

type Bandwidth struct {
	Logger lager.Logger

	ReportAvgBandwidth func(float64)
}

func (h *Bandwidth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.Logger.Session("handle-bandwidth")
	defer logger.Debug("done")

	hasher := sha256.New()
	startTime := time.Now()

	var err error
	result := science.BandwidthExperimentResult{}
	result.NumBytes, err = io.Copy(hasher, r.Body)
	if err != nil {
		logger.Error("read-request-body", err)
		w.WriteHeader(http.StatusInternalServerError)
		encodeError(w, "read-request-failed")
		return
	}

	result.DurationSeconds = time.Since(startTime).Seconds()
	result.SHA256 = hex.EncodeToString(hasher.Sum(nil))
	result.AvgBandwidth = float64(result.NumBytes) / result.DurationSeconds

	logger.Info("stats", lager.Data{"result": result})
	h.ReportAvgBandwidth(result.AvgBandwidth)

	json.NewEncoder(w).Encode(result)
}
