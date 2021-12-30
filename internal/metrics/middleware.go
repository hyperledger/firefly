package metrics


import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelCode   = "code"
	labelMethod = "method"
	labelHost   = "host"
	labelRoute  = "route"
)

// Instrumentation implements the mux middleware and contains configuration options
type Instrumentation struct {
	UseRouteTemplate   bool
	ReqDurationBuckets []float64
	Namespace          string
	Subsystem          string
	Labels             map[string]string
	Registerer         prometheus.Registerer
	reqTotal           *prometheus.CounterVec
	reqSizeBytes       *prometheus.SummaryVec
	reqDurationSecs    *prometheus.HistogramVec
	resSizeBytes       *prometheus.SummaryVec
}

// NewDefaultInstrumentation returns an instrumentation with the default options
func NewDefaultInstrumentation() *Instrumentation {
	i := Instrumentation{
		UseRouteTemplate:   true,
		Namespace:          "mux",
		Subsystem:          "router",
		ReqDurationBuckets: prometheus.DefBuckets,
		Registerer:         prometheus.DefaultRegisterer,
	}

	i.initMetrics()
	return &i
}

// NewCustomInstrumentation returns an instrumentation with custom options
func NewCustomInstrumentation(useRouteTemplate bool, namespace string, subsystem string, reqDurationBuckets []float64, labels map[string]string, registerer prometheus.Registerer) *Instrumentation {
	i := Instrumentation{
		UseRouteTemplate:   useRouteTemplate,
		Namespace:          namespace,
		Subsystem:          subsystem,
		ReqDurationBuckets: reqDurationBuckets,
		Labels:             labels,
		Registerer:         registerer,
	}

	i.initMetrics()
	return &i
}

// Middleware satisifies the mux middleware interface
func (i *Instrumentation) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		sResponseWriter := statusResponseWriter{ResponseWriter: w}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(&sResponseWriter, r)

		defaultLabelVals := []string{fmt.Sprintf("%d", sResponseWriter.status), r.Method, r.Host, i.getRoute(r)}

		i.reqSizeBytes.WithLabelValues(defaultLabelVals...).Observe(float64(estimateRequestSize(r)))
		i.reqTotal.WithLabelValues(defaultLabelVals...).Inc()
		i.resSizeBytes.WithLabelValues(defaultLabelVals...).Observe(float64(sResponseWriter.size))
		i.reqDurationSecs.WithLabelValues(defaultLabelVals...).Observe(time.Now().Sub(startTime).Seconds())
	})
}

// initMetrics initializes all the prometheus metrics
func (i *Instrumentation) initMetrics() {
	i.reqTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "requests_total",
		Subsystem: i.Subsystem,
		Namespace: i.Namespace,
		Help:      "The total number of requests received",
	}, []string{labelCode, labelMethod, labelHost, labelRoute})

	i.reqSizeBytes = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:      "request_size_bytes",
		Subsystem: i.Subsystem,
		Namespace: i.Namespace,
		Help:      "Summary of request bytes received",
	}, []string{labelCode, labelMethod, labelHost, labelRoute})

	i.reqDurationSecs = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:      "request_duration_seconds",
		Subsystem: i.Subsystem,
		Namespace: i.Namespace,
		Help:      "Histogram of the request duration",
		Buckets:   i.ReqDurationBuckets,
	}, []string{labelCode, labelMethod, labelHost, labelRoute})

	i.resSizeBytes = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:      "response_size_bytes",
		Subsystem: i.Subsystem,
		Namespace: i.Namespace,
		Help:      "Summary of response bytes sent",
	}, []string{labelCode, labelMethod, labelHost, labelRoute})

	reg := prometheus.WrapRegistererWith(i.Labels, i.Registerer)
	reg.MustRegister(
		i.reqTotal,
		i.reqSizeBytes,
		i.reqDurationSecs,
		i.resSizeBytes,
	)
}

// getRoute returns the route either as template or the actual url path based on the instrumentation settings
func (i *Instrumentation) getRoute(r *http.Request) string {
	if i.UseRouteTemplate {
		path, _ := mux.CurrentRoute(r).GetPathTemplate()
		return path
	}
	return r.RequestURI
}

// estimateRequestSize approximates the size of the request according to the definition of nginx https://nginx.org/en/docs/http/ngx_http_log_module.html
// request length (including request line, header, and request body). As we want to avoid reqding the request body of every request we will use the value of content length if available
func estimateRequestSize(r *http.Request) int64 {
	var reqSize int64

	// estimate request line https://www.w3.org/Protocols/rfc2616/rfc2616-sec5.html
	reqSize += int64(len(r.Method))
	if r.URL != nil {
		reqSize += int64(len(r.URL.Path))
	}
	reqSize += int64(len(r.Proto))
	reqSize += 4 //SP SP CRLF

	// TODO: needs furhter work to improve estimation
	for key, vals := range r.Header {
		reqSize += int64(len(key))

		for _, v := range vals {
			reqSize += int64(len(v))
		}
		reqSize += 2 // CRLF
	}

	if r.ContentLength != -1 {
		reqSize += r.ContentLength
	}

	return reqSize
}
