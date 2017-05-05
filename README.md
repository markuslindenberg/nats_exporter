# NATS exporter for Prometheus

This is a simple [Prometheus](https://prometheus.io/) exporter that collects metrics from [NATS Server](http://nats.io/)'s HTTP monitoring interface.

By default nats_exporter listens on port 9148 for HTTP requests.

## Installation

### Using `go get`

```bash
go get github.com/markuslindenberg/nats_exporter
```
### Using Docker

```
docker build -t nats_exporter .
docker run --rm -p 9148:9148 nats_exporter -nats.scrape-uri http://gnatsd:8222/
```

## Running

Help on flags:
```
nats_exporter --help

Usage of nats_exporter:
  -log.format value
    	Set the log target and format. Example: "logger:syslog?appname=bob&local=7" or "logger:stdout?json=true" (default "logger:stderr")
  -log.level value
    	Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]
  -nats.scrape-uri string
    	Base URI on which to scrape nats server. (default "http://localhost:8222/")
  -nats.timeout duration
    	Timeout for trying to get stats from nats server. (default 5s)
  -web.listen-address string
    	Address to listen on for web interface and telemetry. (default ":9148")
  -web.telemetry-path string
    	Path under which to expose metrics. (default "/metrics")
```


## Metrics

```
nats_up
nats_exporter_total_scrapes
nats_exporter_json_parse_failures
nats_bytes_in
nats_bytes_out
nats_connections
nats_connections_total
nats_cpu
nats_mem
nats_msgs_in
nats_msgs_out
nats_remotes
nats_routes
nats_server_start
nats_slow_consumers
nats_subscriptions_cache
nats_subscriptions_cache_hit_rate
nats_subscriptions_fanout_avg
nats_subscriptions_fanout_max
nats_subscriptions_inserts
nats_subscriptions_matches
nats_subscriptions_removes
nats_subscriptions_total
```
