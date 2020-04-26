package sqlcon

import (
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestParseConnectionOptions(t *testing.T) {
	defaultMaxConns, defaultMaxIdleConns, defaultMaxConnLifetime := maxParallelism()*2, maxParallelism(), time.Duration(0)
	logger := logrus.New()
	for i, tc := range []struct {
		name, dsn, cleanedDSN  string
		maxConns, maxIdleConns int
		maxConnLifetime        time.Duration
	}{
		{
			name:            "no parameters",
			dsn:             "postgres://user:pwd@host:port",
			cleanedDSN:      "postgres://user:pwd@host:port",
			maxConns:        defaultMaxConns,
			maxIdleConns:    defaultMaxIdleConns,
			maxConnLifetime: defaultMaxConnLifetime,
		},
		{
			name:            "only other parameters",
			dsn:             "postgres://user:pwd@host:port?bar=value&foo=other_value",
			cleanedDSN:      "postgres://user:pwd@host:port?bar=value&foo=other_value",
			maxConns:        defaultMaxConns,
			maxIdleConns:    defaultMaxIdleConns,
			maxConnLifetime: defaultMaxConnLifetime,
		},
		{
			name:            "only maxConns",
			dsn:             "postgres://user:pwd@host:port?max_conns=5254",
			cleanedDSN:      "postgres://user:pwd@host:port?",
			maxConns:        5254,
			maxIdleConns:    defaultMaxIdleConns,
			maxConnLifetime: defaultMaxConnLifetime,
		},
		{
			name:            "only maxIdleConns",
			dsn:             "postgres://user:pwd@host:port?max_idle_conns=9342",
			cleanedDSN:      "postgres://user:pwd@host:port?",
			maxConns:        defaultMaxConns,
			maxIdleConns:    9342,
			maxConnLifetime: defaultMaxConnLifetime,
		},
		{
			name:            "only maxConnLifetime",
			dsn:             "postgres://user:pwd@host:port?max_conn_lifetime=112s",
			cleanedDSN:      "postgres://user:pwd@host:port?",
			maxConns:        defaultMaxConns,
			maxIdleConns:    defaultMaxIdleConns,
			maxConnLifetime: 112 * time.Second,
		},
		{
			name:            "all parameters and others",
			dsn:             "postgres://user:pwd@host:port?max_conns=5254&max_idle_conns=9342&max_conn_lifetime=112s&bar=value&foo=other_value",
			cleanedDSN:      "postgres://user:pwd@host:port?bar=value&foo=other_value",
			maxConns:        5254,
			maxIdleConns:    9342,
			maxConnLifetime: 112 * time.Second,
		},
	} {
		t.Run(fmt.Sprintf("case=%d/name=%s", i, tc.name), func(t *testing.T) {
			maxConns, maxIdleConns, maxConnLifetime, cleanedDSN := ParseConnectionOptions(logger, tc.dsn)
			assert.Equal(t, tc.maxConns, maxConns)
			assert.Equal(t, tc.maxIdleConns, maxIdleConns)
			assert.Equal(t, tc.maxConnLifetime, maxConnLifetime)
			assert.Equal(t, tc.cleanedDSN, cleanedDSN)
		})
	}
}
