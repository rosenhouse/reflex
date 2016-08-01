package handler

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"code.cloudfoundry.org/lager"

	"github.com/rosenhouse/reflex/peer"
)

type PeerList struct {
	Logger lager.Logger
	Peers  peer.List
}

func (h *PeerList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.Logger.Session("handle-list")
	defer logger.Debug("done")

	snapshot := h.Peers.Snapshot(logger)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

type PeerPost struct {
	Logger      lager.Logger
	Peers       peer.List
	AllowedCIDR *net.IPNet
}

func encodeError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{msg})
}

func (h *PeerPost) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.Logger.Session("handle-post")
	defer logger.Debug("done")

	clientIP, err := parseHostIP(r.RemoteAddr) // http server sets r.RemoteAddr to "IP:port"
	if err != nil {
		logger.Error("parse-remote-addr", err, lager.Data{"remote-addr": r.RemoteAddr})
		w.WriteHeader(http.StatusInternalServerError)
		encodeError(w, "cannot parse remote address")
		return
	}

	if !h.AllowedCIDR.Contains(clientIP) {
		logger.Info("peer-not-allowed", lager.Data{"remote-addr": r.RemoteAddr})
		w.WriteHeader(http.StatusForbidden)
		encodeError(w, "source ip not allowed")
		return
	}

	h.Peers.Upsert(logger, clientIP.String())

	snapshot := h.Peers.Snapshot(logger)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

func parseHostIP(addr string) (net.IP, error) {
	// addr might be an ip6 including :'s, so we need to find the _last_ :
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return nil, errors.New("unable to parse address")
	}
	ip := net.ParseIP(addr[:i])
	if ip == nil {
		return nil, errors.New("cannot parse as ip")
	}
	return ip, nil
}
