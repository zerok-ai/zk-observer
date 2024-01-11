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
