/*
 * Copyright (c) 2022, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package metric is the package to report the metrics to Prometheus
package metric

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	log "github.com/sirupsen/logrus"
	"github.com/wfusion/gofusion/common/utils"
)

const module = "Metric"

// MetricsType is the generic type of metrics
type MetricsType interface {
	*prometheus.CounterVec | *prometheus.GaugeVec | *prometheus.HistogramVec | *prometheus.SummaryVec
}

var (
	registries   = make([]*prometheus.Registry, 0)
	counterMap   = make(map[string]*prometheus.CounterVec)
	gaugeMap     = make(map[string]*prometheus.GaugeVec)
	histogramMap = make(map[string]*prometheus.HistogramVec)
	summaryMap   = make(map[string]*prometheus.SummaryVec)

	rwlock = new(sync.RWMutex)
)

var (
	validMetric = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)
	validLabel  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// Counter get the counter metric by key
func Counter(key string) *prometheus.CounterVec {
	rwlock.RLock()
	defer rwlock.RUnlock()
	return counterMap[key]
}

// Gauge get the gauge metric by key
func Gauge(key string) *prometheus.GaugeVec {
	rwlock.RLock()
	defer rwlock.RUnlock()
	return gaugeMap[key]
}

// NewCounter create the counter metric
func NewCounter(namespace, subsystem, name, metric string,
	help string, labels []string, constLabels prometheus.Labels) *prometheus.CounterVec {
	rwlock.Lock()
	defer rwlock.Unlock()

	metricName, err := getAndValid(namespace, subsystem, name, metric, labels, constLabels)
	if err != nil {
		log.Errorf("[namespace: %s, subsystem: %s, name: %s, metric: %s] %v",
			namespace, subsystem, name, metric, err)
		return nil
	}

	if m, find := counterMap[metricName]; find {
		log.Debugf("[%s] Counter <%s> already created!", module, metricName)
		return m
	}

	counterMap[metricName] = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metricName,
			Help: help,
		},
		mergeLabels(labels, constLabels),
	)

	prometheus.MustRegister(counterMap[metricName])
	log.Infof("[%s] Counter <%s> is created!", module, metricName)
	return counterMap[metricName]
}

// NewGauge create the gauge metric
func NewGauge(namespace, subsystem, name, metric string,
	help string, labels []string, constLabels prometheus.Labels) *prometheus.GaugeVec {
	rwlock.Lock()
	defer rwlock.Unlock()

	metricName, err := getAndValid(namespace, subsystem, name, metric, labels, constLabels)
	if err != nil {
		log.Errorf("[%s] %v", module, err)
		return nil
	}

	if m, find := gaugeMap[metricName]; find {
		log.Debugf("[%s] Gauge <%s> already created!", module, metricName)
		return m
	}

	gaugeMap[metricName] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: metricName,
			Help: help,
		},
		mergeLabels(labels, constLabels),
	)

	prometheus.MustRegister(gaugeMap[metricName])

	log.Infof("[%s] Gauge <%s> is created!", module, metricName)
	return gaugeMap[metricName]
}

func mergeLabels(labels []string, constLabels prometheus.Labels) []string {
	l := make([]string, 0, len(labels)+len(constLabels))
	l = append(l, labels...)

	for labelName := range constLabels {
		l = append(l, labelName)
	}

	return l
}

func getAndValid(namespace, subsystem, name, metric string, labels []string, constLabels prometheus.Labels) (string, error) {
	metricName := GetName(namespace, subsystem, name, metric)
	if ValidMetricName(metricName) == false {
		return "", fmt.Errorf("invalid metric name: %s", metricName)
	}

	for _, l := range labels {
		if ValidLabelName(l) == false {
			return "", fmt.Errorf("invalid label name: %s", l)
		}
	}

	for l := range constLabels {
		if !ValidLabelName(l) {
			return "", fmt.Errorf("invalid const label name: %s", l)
		}
	}

	for _, l := range labels {
		if _, ok := constLabels[l]; ok {
			return "", fmt.Errorf("label '%s' is duplicated", l)
		}
	}

	return metricName, nil
}

// GetName generate the metric key by a number of strings
func GetName(fields ...string) string {
	name := ""
	for _, v := range fields {
		v = RemoveInvalidChars(v)
		if len(v) > 0 {
			name += v + "_"
		}
	}

	if len(name) > 0 && name[len(name)-1] == '_' {
		name = name[:len(name)-1]
	}

	log.Debugf("[%s] get the name: %s", module, name)
	return name
}

// ValidMetricName check if the metric name is valid
func ValidMetricName(name string) bool {
	return validMetric.MatchString(name)
}

// ValidLabelName check if the label name is valid
func ValidLabelName(label string) bool {
	return validLabel.MatchString(label)
}

// ValidMetricChar check if the char is valid for metric name
func ValidMetricChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_' || ch == ':'
}

// RemoveInvalidChars remove invalid chars
func RemoveInvalidChars(name string) string {
	var result []byte
	i := 0

	// skip all of the non-alphabetic chars
	for ; i < len(name); i++ {
		ch := name[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			break
		}
	}

	// remove the invalid chars
	for ; i < len(name); i++ {
		if ValidMetricChar(name[i]) {
			result = append(result, name[i])
		}
	}
	return string(result)
}

// AddConstLabels append user defined labels in the configuration file to the
// predefined label set.
func AddConstLabels(labels prometheus.Labels, constLabels prometheus.Labels) prometheus.Labels {
	for k, v := range constLabels {
		labels[k] = v
	}
	return labels
}

type PrometheusPushSink struct {
	namespace    string
	address      string
	pusher       *push.Pusher
	stopChan     chan bool
	pushInterval time.Duration
}

func NewPrometheusPushSink(namespace, address string, pushInterval time.Duration, done chan bool) *PrometheusPushSink {
	sink := &PrometheusPushSink{
		namespace:    namespace,
		address:      address,
		pushInterval: pushInterval,
		stopChan:     done,
	}
	sink.pusher = push.New(address, namespace).Collector(sink)
	sink.flushMetrics()
	return sink
}

// Describe sends a Collector.Describe value from the descriptor created around PrometheusSink.Name
// Note that we cannot describe all the metrics (gauges, counters, summaries) in the sink as
// metrics can be added at any point during the lifecycle of the sink, which does not respect
// the idempotency aspect of the Collector.Describe() interface
func (p *PrometheusPushSink) Describe(c chan<- *prometheus.Desc) {
	// dummy value to be able to register and unregister "empty" sinks
	// Note this is not actually retained in the PrometheusSink so this has no side effects
	// on the caller's sink. So it shouldn't show up to any of its consumers.
	prometheus.NewGauge(prometheus.GaugeOpts{Name: p.namespace, Help: p.namespace}).Describe(c)
}

// Collect meets the collection interface and allows us to enforce our expiration
// logic to clean up ephemeral metrics if their value haven't been set for a
// duration exceeding our allowed expiration time.
func (p *PrometheusPushSink) Collect(c chan<- prometheus.Metric) {
	p.collectAtTime(c, time.Now())
}

func (p *PrometheusPushSink) collectAtTime(c chan<- prometheus.Metric, t time.Time) {
	rwlock.RLock()
	defer rwlock.RUnlock()
	for _, gauge := range gaugeMap {
		gauge.Collect(c)
	}
	for _, counter := range counterMap {
		counter.Collect(c)
	}
	for _, histogram := range histogramMap {
		histogram.Collect(c)
	}
	for _, summary := range summaryMap {
		summary.Collect(c)
	}
}

func (p *PrometheusPushSink) flushMetrics() {
	go func() {
		ticker := time.NewTicker(p.pushInterval)
		defer ticker.Stop()

		err := retry.Retry(func(attempt uint) (err error) {
			_, err = utils.Catch(func() {
				for {
					select {
					case ti := <-ticker.C:
						if err := p.pusher.Push(); err != nil {
							log.Errorf("[%s] Prometheus push to prometheus err at %s: %s\n",
								module, ti, err)
						}
					case <-p.stopChan:
						log.Infof("[%s] Prometheus push cycle exited\n", module)
						return
					}
				}
			})
			return
		}, strategy.Limit(86400)) // 24 * 60 * 60 * s.pushInterval
		if err != nil {
			log.Errorf("[%s] Prometheus exit unexpectedly: %s\n", module, err)
		}
	}()
}
