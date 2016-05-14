package main

import (
	"time"
)

type Varz struct {
	Version          string             `json:"version"`
	Start            time.Time          `json:"start"`
	Mem              float64            `json:"mem"`
	Connections      float64            `json:"connections"`
	TotalConnections float64            `json:"total_connections"`
	Routes           float64            `json:"routes"`
	Remotes          float64            `json:"remotes"`
	InMsgs           float64            `json:"in_msgs"`
	OutMsgs          float64            `json:"out_msgs"`
	InBytes          float64            `json:"in_bytes"`
	OutBytes         float64            `json:"out_bytes"`
	SlowConsumers    float64            `json:"slow_consumers"`
	HTTPReqStats     map[string]float64 `json:"http_req_stats"`
}

type Subsz struct {
	NumSubscriptions float64 `json:"num_subscriptions"`
	NumCache         float64 `json:"num_cache"`
	NumInserts       float64 `json:"num_inserts"`
	NumRemoves       float64 `json:"num_removes"`
	NumMatches       float64 `json:"num_matches"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	MaxFanout        float64 `json:"max_fanout"`
	AvgFanout        float64 `json:"avg_fanout"`
}
