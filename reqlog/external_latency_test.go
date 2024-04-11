// Copyright © 2024 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package reqlog

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"golang.org/x/sync/errgroup"

	"github.com/ory/x/assertx"
)

func TestExternalLatencyMiddleware(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ExternalCallsMiddleware(w, r, func(w http.ResponseWriter, r *http.Request) {
			var (
				wg         sync.WaitGroup
				res0, res1 string
				err        error
			)

			wg.Add(3)
			go func() {
				res0 = MeasureExternalCall(r.Context(), "", "", func() string {
					time.Sleep(100 * time.Millisecond)
					return "foo"
				})
				wg.Done()
			}()
			go func() {
				res1, err = MeasureExternalCallErr(r.Context(), "", "", func() (string, error) {
					time.Sleep(100 * time.Millisecond)
					return "bar", nil
				})
				wg.Done()
			}()
			go func() {
				_ = MeasureExternalCall(WithDisableExternalLatencyMeasurement(r.Context()), "", "", func() error {
					time.Sleep(100 * time.Millisecond)
					return nil
				})
				wg.Done()
			}()
			wg.Wait()
			total := TotalExternalLatency(r.Context())
			_ = json.NewEncoder(w).Encode(map[string]any{
				"res0":  res0,
				"res1":  res1,
				"err":   err,
				"total": total,
			})
		})
	}))
	defer ts.Close()

	bodies := make([][]byte, 100)
	eg := errgroup.Group{}
	for i := range bodies {
		eg.Go(func() error {
			res, err := http.Get(ts.URL)
			if err != nil {
				return err
			}
			defer res.Body.Close()
			bodies[i], err = io.ReadAll(res.Body)
			if err != nil {
				return err
			}
			return nil
		})
	}

	require.NoError(t, eg.Wait())

	for _, body := range bodies {
		assertx.EqualAsJSONExcept(t, map[string]any{
			"res0": "foo",
			"res1": "bar",
			"err":  nil,
		}, json.RawMessage(body), []string{"total"})

		actualTotal := gjson.GetBytes(body, "total").Int()
		assert.GreaterOrEqual(t, actualTotal, int64(200*time.Millisecond), string(body))
	}
}
