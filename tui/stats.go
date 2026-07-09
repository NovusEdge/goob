package main

import (
	"encoding/json"
	"net/http"
	"time"
)

const statsURL = "http://127.0.0.1:8787/stats"

// Stats mirrors the daemon's GET /stats payload.
type Stats struct {
	Model   string  `json:"model"`
	Ticks   int     `json:"ticks"`
	Spend   float64 `json:"spend_usd"`
	Latency float64 `json:"last_latency_ms"`
}

func fetchStats() (Stats, error) {
	var st Stats
	client := http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get(statsURL)
	if err != nil {
		return st, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&st)
	return st, err
}
