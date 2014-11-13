package pipe

import (
	"encoding/json"
	"time"
)

type eventKind int

func (e eventKind) String() string {
	switch e {
	case bootKind:
		return "boot"
	case errorKind:
		return "error"
	case metricsKind:
		return "metrics"
	}
	return ""
}

const (
	bootKind eventKind = iota
	errorKind
	metricsKind
)

/*
 * Events
 */
type event struct {
	Ts           int64  `json:"ts"`
	Kind         string `json:"event"`
	bootEvent    `json:",omitempty"`
	metricsEvent `json:",omitempty"`
}

func (e event) String() string {
	ba, _ := json.Marshal(e)
	return string(ba)
}

/*
 * bootevents are sent when the pipeline has been started
 */
type bootEvent struct {
	Version   string            `json:"version,omitempty"`
	Endpoints map[string]string `json:"endpoints,omitempty"`
}

func NewBootEvent(ts int64, version string, endpoints map[string]string) event {
	e := event{Ts: ts, Kind: bootKind.String()}
	e.Version = version
	e.Endpoints = endpoints
	return e
}

/*
 * Metrics events are sent by the nodes periodically
 */
type metricsEvent struct {
	Path       string `json:"path,omitempty"`
	RecordsIn  int    `json:"records_in,omitempty"`
	RecordsOut int    `json:"records_out,omitempty"`
}

func NewMetricsEvent(ts int64, path string, in, out int) event {
	e := event{Ts: ts, Kind: metricsKind.String()}
	e.Path = path
	e.RecordsIn = in
	e.RecordsOut = out
	return e
}

/*
 * lets keep track of metrics on a nodeimpl, and send them out periodically to our event chan
 */
type nodeMetrics struct {
	ticker     *time.Ticker
	eChan      chan event
	path       string
	RecordsIn  int
	RecordsOut int
}

func NewNodeMetrics(path string, eventChan chan event, interval time.Duration) *nodeMetrics {
	m := &nodeMetrics{path: path, eChan: eventChan}

	// if we have a non zero interval then spawn a ticker to send metrics out the channel
	if interval > 0 {
		m.ticker = time.NewTicker(interval)
		go func() {
			for _ = range m.ticker.C {
				m.Send()
			}
		}()
	}
	return m
}

func (m *nodeMetrics) Send() {
	m.eChan <- NewMetricsEvent(time.Now().Unix(), m.path, m.RecordsIn, m.RecordsOut)
}

func (m *nodeMetrics) Stop() {
	if m.ticker != nil {
		m.ticker.Stop()
	}
}