package science

type BandwidthExperimentResult struct {
	NumBytes        int64   `json:"num_bytes"`
	DurationSeconds float64 `json:"duration_seconds"`
	AvgBandwidth    float64 `json:"avg_bandwidth"`
	SHA256          string  `json:"sha256"`
}
