package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/segmentio/analytics-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	"github.com/ory/x/metricsx"
)

func main() {
	wk := os.Getenv("WRITE_KEY")
	if wk == "" {
		wk = "foo"
	}

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v\n\t%s", r, out)
	}))

	defer api.Close()

	n := negroni.New()
	segmentMiddleware := metricsx.NewMetricsManagerWithConfig(
		"foo",
		true,
		wk,
		[]string{},
		logrus.New(),
		"metrics-middleware",
		1.0,
		analytics.Config{
			Interval:  time.Second,
			BatchSize: 1,
			Endpoint:  "https://ory-metrics-server.herokuapp.com",
		},
	)
	go segmentMiddleware.RegisterSegment("1.0.0", "c1b", time.Now().String())
	go segmentMiddleware.CommitMemoryStatistics()
	n.Use(segmentMiddleware)
	r := httprouter.New()
	r.GET("/", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		writer.WriteHeader(http.StatusNoContent)
	})
	n.UseHandler(r)

	ts := httptest.NewServer(n)
	defer ts.Close()

	printMemUsage()

	go func() {
		for {
			printMemUsage()
			time.Sleep(time.Second)
		}
	}()

	c := ts.Client()
	for i := 0; i <= 5000; i++ {
		func() {
			resp, err := c.Get(ts.URL)
			if err != nil {
				logrus.WithError(err).Fatalf("Unable to get")
			}
			defer resp.Body.Close()

			if http.StatusNoContent != resp.StatusCode {
				logrus.WithError(err).Fatalf("Unable to get")
			}
			time.Sleep(time.Millisecond * 10)
		}()
	}

	printMemUsage()
}

func printMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
