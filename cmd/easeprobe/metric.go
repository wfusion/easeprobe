package main

import (
	"time"

	"github.com/wfusion/easeprobe/conf"
	"github.com/wfusion/easeprobe/metric"
)

func runMetric(done chan bool) {
	if conf.Get().Settings.Prometheus.Mode != conf.PrometheusModePush {
		return
	}

	c := conf.Get()
	interval, _ := time.ParseDuration(c.Settings.Prometheus.PushInterval)
	if interval == 0 {
		interval = 30 * time.Second
	}
	metric.NewPrometheusPushSink(c.Settings.Name, c.Settings.Prometheus.Addr, interval, done)
}
