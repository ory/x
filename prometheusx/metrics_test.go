package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	prometheus "github.com/ory/x/prometheusx"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ioprometheusclient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
	"github.com/urfave/negroni"
)

func TestMetrics(t *testing.T) {
	testApp := "test_app"
	testPath := "/test/path"

	n := negroni.New()
	handler := func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		prometheus.NewMetrics(testApp, prometheus.HTTPMetrics, "", "", "").Instrument(rw, next, r.RequestURI)(rw, r)
	}
	n.UseFunc(handler)

	router := httprouter.New()
	router.GET(testPath, func(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
		rw.WriteHeader(http.StatusBadRequest)
	})
	router.GET(prometheus.MetricsPrometheusPath, func(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
		promhttp.Handler().ServeHTTP(rw, r)
	})
	n.UseHandler(router)

	ts := httptest.NewServer(n)
	defer ts.Close()

	resp, err := http.Get(ts.URL + testPath)
	require.NoError(t, err)
	require.EqualValues(t, http.StatusBadRequest, resp.StatusCode)

	promresp, err := http.Get(ts.URL + prometheus.MetricsPrometheusPath)
	require.NoError(t, err)
	require.EqualValues(t, http.StatusOK, promresp.StatusCode)

	textParser := expfmt.TextParser{}
	text, err := textParser.TextToMetricFamilies(promresp.Body)
	require.NoError(t, err)
	require.EqualValues(t, "http_response_time_seconds", *text["http_response_time_seconds"].Name)
	require.EqualValues(t, testPath, getLabelValue("endpoint", text["http_response_time_seconds"].Metric))
	require.EqualValues(t, testApp, getLabelValue("app", text["http_response_time_seconds"].Metric))

	require.EqualValues(t, "http_requests_total", *text["http_requests_total"].Name)
	require.EqualValues(t, "400", getLabelValue("code", text["http_requests_total"].Metric))
	require.EqualValues(t, testApp, getLabelValue("app", text["http_requests_total"].Metric))

	require.EqualValues(t, "http_requests_duration_seconds", *text["http_requests_duration_seconds"].Name)
	require.EqualValues(t, "400", getLabelValue("code", text["http_requests_duration_seconds"].Metric))
	require.EqualValues(t, testApp, getLabelValue("app", text["http_requests_duration_seconds"].Metric))

	require.EqualValues(t, "http_response_size_bytes", *text["http_response_size_bytes"].Name)
	require.EqualValues(t, "400", getLabelValue("code", text["http_response_size_bytes"].Metric))
	require.EqualValues(t, testApp, getLabelValue("app", text["http_response_size_bytes"].Metric))

	require.EqualValues(t, "http_requests_size_bytes", *text["http_requests_size_bytes"].Name)
	require.EqualValues(t, "400", getLabelValue("code", text["http_requests_size_bytes"].Metric))
	require.EqualValues(t, testApp, getLabelValue("app", text["http_requests_size_bytes"].Metric))

	require.EqualValues(t, "http_requests_statuses_total", *text["http_requests_statuses_total"].Name)
	require.EqualValues(t, "4xx", getLabelValue("status_bucket", text["http_requests_statuses_total"].Metric))
	require.EqualValues(t, testApp, getLabelValue("app", text["http_requests_statuses_total"].Metric))
}

func getLabelValue(name string, metric []*ioprometheusclient.Metric) string {
	for _, label := range metric[0].Label {
		if *label.Name == name {
			return *label.Value
		}
	}

	return ""
}
