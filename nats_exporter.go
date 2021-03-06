// Copyright 2016 Markus Lindenberg
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

const (
	namespace = "nats"
)

var (
	upLabelNames       = []string{"version"}
	requestsLabelNames = []string{"path"}
)

// Exporter collects gnatsd stats from the given URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	VarzURI, SubszURI string
	mutex             sync.RWMutex
	client            *http.Client

	totalScrapes, jsonParseFailures                                  prometheus.Counter
	up                                                               prometheus.Gauge
	startTime, cpu, mem, connections, routes, remotes, slowConsumers prometheus.Gauge
	totalConnections, inMsgs, outMsgs, inBytes, outBytes             prometheus.Gauge
	httpRequests                                                     *prometheus.GaugeVec
	numSubscriptions, numCache, numInserts, numRemoves, numMatches   prometheus.Gauge
	cacheHitRate, maxFanout, avgFanout                               prometheus.Gauge
}

// NewExporter returns an initialized Exporter.
func NewExporter(baseURI string, timeout time.Duration) *Exporter {
	return &Exporter{
		VarzURI:  strings.TrimRight(baseURI, "/") + "/varz",
		SubszURI: strings.TrimRight(baseURI, "/") + "/subsz",

		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the last scrape of Nats Server successful.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_total_scrapes",
			Help:      "Current total Nats Server scrapes.",
		}),
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_json_parse_failures",
			Help:      "Number of errors while parsing JSON.",
		}),

		startTime: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "server_start",
			Help:      "Timestamp of Nats Server startup.",
		}),
		mem: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "mem",
			Help:      "mem",
		}),
		cpu: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cpu",
			Help:      "cpu",
		}),
		connections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "connections",
			Help:      "connections",
		}),
		totalConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "connections_total",
			Help:      "connections_total",
		}),
		routes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "routes",
			Help:      "routes",
		}),
		remotes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "remotes",
			Help:      "remotes",
		}),
		inMsgs: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "msgs_in",
			Help:      "msgs_in",
		}),
		outMsgs: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "msgs_out",
			Help:      "msgs_out",
		}),
		inBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "bytes_in",
			Help:      "bytes_in",
		}),
		outBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "bytes_out",
			Help:      "bytes_out",
		}),
		slowConsumers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "slow_consumers",
			Help:      "slow_consumers",
		}),
		httpRequests: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "http_requests",
			Help:      "http_requests",
		}, requestsLabelNames),

		numSubscriptions: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "subscriptions",
			Name:      "total",
			Help:      "subscriptions_total",
		}),
		numCache: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "subscriptions",
			Name:      "cache",
			Help:      "subscriptions_cache",
		}),
		numInserts: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "subscriptions",
			Name:      "inserts",
			Help:      "subscription_inserts",
		}),
		numRemoves: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "subscriptions",
			Name:      "removes",
			Help:      "subscription_removes",
		}),
		numMatches: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "subscriptions",
			Name:      "matches",
			Help:      "subscription_matches",
		}),
		cacheHitRate: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "subscriptions",
			Name:      "cache_hit_rate",
			Help:      "subscription_cache_hit_rate",
		}),
		maxFanout: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "subscriptions",
			Name:      "fanout_max",
			Help:      "subscription_fanout_max",
		}),
		avgFanout: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "subscriptions",
			Name:      "fanout_avg",
			Help:      "subscription_fanout_avg",
		}),

		client: &http.Client{
			Transport: &http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					c, err := net.DialTimeout(netw, addr, timeout)
					if err != nil {
						return nil, err
					}
					if err := c.SetDeadline(time.Now().Add(timeout)); err != nil {
						return nil, err
					}
					return c, nil
				},
			},
		},
	}
}

// Describe describes all the metrics ever exported by the NATS exporter.
// It implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up.Desc()
	ch <- e.totalScrapes.Desc()
	ch <- e.jsonParseFailures.Desc()

	ch <- e.startTime.Desc()
	ch <- e.mem.Desc()
	ch <- e.cpu.Desc()
	ch <- e.connections.Desc()
	ch <- e.totalConnections.Desc()
	ch <- e.routes.Desc()
	ch <- e.remotes.Desc()
	ch <- e.inMsgs.Desc()
	ch <- e.outMsgs.Desc()
	ch <- e.inBytes.Desc()
	ch <- e.outBytes.Desc()
	ch <- e.slowConsumers.Desc()
	e.httpRequests.Describe(ch)

	ch <- e.numSubscriptions.Desc()
	ch <- e.numCache.Desc()
	ch <- e.numInserts.Desc()
	ch <- e.numRemoves.Desc()
	ch <- e.numMatches.Desc()
	ch <- e.cacheHitRate.Desc()
	ch <- e.maxFanout.Desc()
	ch <- e.avgFanout.Desc()
}

// Collect fetches the stats from gnatsd and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	e.httpRequests.Reset()
	e.scrape()

	ch <- e.up
	ch <- e.totalScrapes
	ch <- e.jsonParseFailures

	ch <- e.startTime
	ch <- e.mem
	ch <- e.cpu
	ch <- e.connections
	ch <- e.totalConnections
	ch <- e.routes
	ch <- e.remotes
	ch <- e.inMsgs
	ch <- e.outMsgs
	ch <- e.inBytes
	ch <- e.outBytes
	ch <- e.slowConsumers
	e.httpRequests.Collect(ch)

	ch <- e.numSubscriptions
	ch <- e.numCache
	ch <- e.numInserts
	ch <- e.numRemoves
	ch <- e.numMatches
	ch <- e.cacheHitRate
	ch <- e.maxFanout
	ch <- e.avgFanout
}

func (e *Exporter) scrape() {
	e.totalScrapes.Inc()
	var err error

	var varz Varz
	err = e.fetch(e.VarzURI, &varz)
	if err != nil {
		e.up.Set(0)
		log.Errorf("Can't scrape varz: %s", err)
		return
	}

	var subsz Subsz
	err = e.fetch(e.SubszURI, &subsz)
	if err != nil {
		e.up.Set(0)
		log.Errorf("Can't scrape subsz: %s", err)
		return
	}

	e.up.Set(1)

	e.startTime.Set(float64(varz.Start.Unix()))
	e.mem.Set(float64(varz.Mem))
	e.cpu.Set(varz.CPU)
	e.connections.Set(float64(varz.Connections))
	e.totalConnections.Set(float64(varz.TotalConnections))
	e.routes.Set(float64(varz.Routes))
	e.remotes.Set(float64(varz.Remotes))
	e.inMsgs.Set(float64(varz.InMsgs))
	e.outMsgs.Set(float64(varz.OutMsgs))
	e.inBytes.Set(float64(varz.InBytes))
	e.outBytes.Set(float64(varz.OutBytes))
	e.slowConsumers.Set(float64(varz.SlowConsumers))
	for path, requests := range varz.HTTPReqStats {
		e.httpRequests.WithLabelValues(path).Set(float64(requests))
	}

	e.numSubscriptions.Set(float64(subsz.NumSubs))
	e.numCache.Set(float64(subsz.NumCache))
	e.numInserts.Set(float64(subsz.NumInserts))
	e.numRemoves.Set(float64(subsz.NumRemoves))
	e.numMatches.Set(float64(subsz.NumMatches))
	e.cacheHitRate.Set(subsz.CacheHitRate)
	e.maxFanout.Set(float64(subsz.MaxFanout))
	e.avgFanout.Set(subsz.AvgFanout)
}

func (e *Exporter) fetch(uri string, v interface{}) error {
	resp, err := e.client.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		e.jsonParseFailures.Inc()
		return fmt.Errorf("Can't read JSON: %v", err)
	}

	return nil
}

func main() {
	var (
		listenAddress = flag.String("web.listen-address", ":9148", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		natsScrapeURI = flag.String("nats.scrape-uri", "http://localhost:8222/", "Base URI on which to scrape nats server.")
		natsTimeout   = flag.Duration("nats.timeout", 5*time.Second, "Timeout for trying to get stats from nats server.")
	)
	flag.Parse()

	// Listen to signals
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGINT)

	exporter := NewExporter(*natsScrapeURI, *natsTimeout)
	prometheus.MustRegister(exporter)

	// Setup HTTP server
	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>NATS Exporter</title></head>
             <body>
             <h1>NATS Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	go func() {
		log.Infof("Starting Server: %s", *listenAddress)
		log.Fatal(http.ListenAndServe(*listenAddress, nil))
	}()

	s := <-sigchan
	log.Infof("Received %v, terminating", s)
	os.Exit(0)
}
