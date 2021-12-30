package metrics


import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gotest.tools/assert"
)

func TestEstimateRequestSize(t *testing.T) {

	type testCase struct {
		req  *http.Request
		size int64
	}

	singleHeaderReq := httptest.NewRequest("GET", "http://example.com/foo", nil)
	singleHeaderReq.Header.Add("User-Agent", "testing")

	twoHeaderReq := httptest.NewRequest("GET", "http://example.com/foo", nil)
	twoHeaderReq.Header.Add("User-Agent", "testing")
	twoHeaderReq.Header.Add("Referer", "https://testing.com")

	postReq := httptest.NewRequest("POST", "http://example.com/foo", nil)
	postReq.Body = ioutil.NopCloser(strings.NewReader("forTesting!"))

	postContentLengthReq := httptest.NewRequest("POST", "http://example.com/foo", nil)
	postContentLengthReq.Header.Add("Content-Length", "11")
	postContentLengthReq.Body = ioutil.NopCloser(strings.NewReader("forTesting!"))

	testCases := []testCase{
		{
			req:  httptest.NewRequest("GET", "https://example.com/foo", nil),
			size: 19,
		},
		{
			req:  httptest.NewRequest("GET", "http://example.com/foo", nil),
			size: 19,
		},
		{
			req:  httptest.NewRequest("PATCH", "http://example.com/foo", nil),
			size: 21,
		},
		{
			req:  singleHeaderReq,
			size: 38,
		},
		{
			req:  twoHeaderReq,
			size: 66,
		},
		{
			req:  postReq,
			size: 20,
		},
		{
			req:  postContentLengthReq,
			size: 38,
		},
	}

	for _, c := range testCases {
		actualSize := estimateRequestSize(c.req)
		assert.Equal(t, c.size, actualSize)
	}
}

func TestEndToEnd(t *testing.T) {

	type testCase struct {
		inst            *Instrumentation
		expMetricsRegex []string
	}

	testCases := []testCase{
		{
			inst: NewDefaultInstrumentation(),
			expMetricsRegex: []string{
				fmt.Sprintf(".*%s_%s_request_duration_seconds_count{code=\"%d\",.*,route=\"/helloworld/{id}\"} %d.*", "mux", "router", http.StatusAccepted, 3),
				fmt.Sprintf(".*%s_%s_requests_total{code=\"%d\",.*,route=\"/helloworld/{id}\"} %d.*", "mux", "router", http.StatusAccepted, 3),
				fmt.Sprintf(".*%s_%s_request_size_bytes_count{code=\"%d\",.*,route=\"/helloworld/{id}\"} %d.*", "mux", "router", http.StatusAccepted, 3),
				fmt.Sprintf(".*%s_%s_response_size_bytes_count{code=\"%d\",.*,route=\"/helloworld/{id}\"} %d.*", "mux", "router", http.StatusAccepted, 3),
			},
		},
		{
			inst: NewCustomInstrumentation(true, "test", "case", prometheus.DefBuckets, map[string]string{}, prometheus.DefaultRegisterer),
			expMetricsRegex: []string{
				fmt.Sprintf(".*%s_%s_request_duration_seconds_count{code=\"%d\",.*,route=\"/helloworld/{id}\"} %d.*", "test", "case", http.StatusAccepted, 3),
				fmt.Sprintf(".*%s_%s_requests_total{code=\"%d\",.*,route=\"/helloworld/{id}\"} %d.*", "test", "case", http.StatusAccepted, 3),
				fmt.Sprintf(".*%s_%s_request_size_bytes_count{code=\"%d\",.*,route=\"/helloworld/{id}\"} %d.*", "test", "case", http.StatusAccepted, 3),
				fmt.Sprintf(".*%s_%s_response_size_bytes_count{code=\"%d\",.*,route=\"/helloworld/{id}\"} %d.*", "test", "case", http.StatusAccepted, 3),
			},
		},
		{
			inst: NewCustomInstrumentation(false, "test", "next", prometheus.DefBuckets, map[string]string{}, prometheus.DefaultRegisterer),
			expMetricsRegex: []string{
				fmt.Sprintf(".*%s_%s_request_duration_seconds_count{code=\"%d\",.*,route=\"/helloworld/1\"} %d.*", "test", "next", http.StatusAccepted, 1),
				fmt.Sprintf(".*%s_%s_requests_total{code=\"%d\",.*,route=\"/helloworld/2\"} %d.*", "test", "next", http.StatusAccepted, 1),
				fmt.Sprintf(".*%s_%s_request_size_bytes_count{code=\"%d\",.*,route=\"/helloworld/3\"} %d.*", "test", "next", http.StatusAccepted, 1),
				fmt.Sprintf(".*%s_%s_response_size_bytes_count{code=\"%d\",.*,route=\"/helloworld/1\"} %d.*", "test", "next", http.StatusAccepted, 1),
			},
		},
	}

	for _, c := range testCases {
		r := mux.NewRouter()
		r.Use(c.inst.Middleware)

		r.Path("/metrics").Handler(promhttp.Handler())
		r.Path("/helloworld/{id}").HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
		)

		s := httptest.NewServer(r)
		defer s.Close()

		_, err := s.Client().Get(fmt.Sprintf("%s/helloworld/1", s.URL))
		assert.NilError(t, err)
		_, err = s.Client().Get(fmt.Sprintf("%s/helloworld/2", s.URL))
		assert.NilError(t, err)
		_, err = s.Client().Get(fmt.Sprintf("%s/helloworld/3", s.URL))
		assert.NilError(t, err)

		resp, err := s.Client().Get(fmt.Sprintf("%s/metrics", s.URL))
		assert.NilError(t, err)

		body, err := ioutil.ReadAll(resp.Body)
		assert.NilError(t, err)

		for _, m := range c.expMetricsRegex {
			r := regexp.MustCompile(m)
			assert.Equal(t, r.Match(body), true, fmt.Sprintf("Regex %s did not match in any of the metrics returned", r.String()))
		}
	}

}
