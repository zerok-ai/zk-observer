package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	//total traces span data requested from receiver
	TotalTracesSpanDataRequestedFromReceiver = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zerok_otlp_receiver_span_details_traces_requested_total",
		Help: "total traces span data requested from receiver.",
	})

	// TotalFetchRequestsFromSM is the total number of fetch requests received from scenario manager.
	TotalFetchRequestsFromSM = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zerok_otlp_receiver_span_details_fetch_requests_total",
		Help: "total fetch calls received from scenario manager.",
	})

	// TotalFetchRequestsFromSMError is the total number of fetch requests received from scenario manager.
	TotalFetchRequestsFromSMError = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zerok_otlp_receiver_span_details_fetch_requests_error",
		Help: "total fetch calls received from scenario manager.",
	})

	// TotalFetchRequestsFromSMSuccess is the total number of fetch requests received from scenario manager.
	TotalFetchRequestsFromSMSuccess = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zerok_otlp_receiver_span_details_fetch_requests_success",
		Help: "total fetch calls received from scenario manager.",
	})

	// TotalFetchRequestsFromSMError is the total number of fetch requests received from scenario manager.
	TotalSpansProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zerok_receiver_spans_processed_total",
		Help: "Total spans processed by the receiver.",
	})

	// TotalSpansFiltered is the total number of spans filtered by the receiver.
	TotalSpansFiltered = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zerok_receiver_spans_filtered_total",
		Help: "Total spans filtered by the receiver.",
	})
)

func BadgerCollector(namespace string) prometheus.Collector {
	exports := map[string]*prometheus.Desc{}
	metricnames := []string{
		"badger_disk_reads_total",
		"badger_disk_writes_total",
		"badger_read_bytes",
		"badger_written_bytes",
		"badger_lsm_level_gets_total",
		"badger_lsm_bloom_hits_total",
		"badger_gets_total",
		"badger_puts_total",
		"badger_blocked_puts_total",
		"badger_memtable_gets_total",
		"badger_lsm_size_bytes",
		"badger_vlog_size_bytes",
		"badger_pending_writes_total",
	}
	for _, name := range metricnames {
		exportname := name
		if exportname != "" {
			exportname = namespace + "_" + exportname
		}
		exports[name] = prometheus.NewDesc(
			exportname,
			"badger db metric "+name,
			nil, nil,
		)
	}
	return prometheus.NewExpvarCollector(exports)
}
