package crdbx

import (
	"github.com/ory/x/urlx"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestConsistencyLevelFromString(t *testing.T) {
	assert.Equal(t, ConsistencyLevelStrong, ConsistencyLevelFromString(""))
	assert.Equal(t, ConsistencyLevelStrong, ConsistencyLevelFromString("strong"))
	assert.Equal(t, ConsistencyLevelEventual, ConsistencyLevelFromString("eventual"))
	assert.Equal(t, ConsistencyLevelStrong, ConsistencyLevelFromString("lol"))
}

func TestConsistencyLevelFromRequest(t *testing.T) {
	assert.Equal(t, ConsistencyLevelStrong, ConsistencyLevelFromRequest(&http.Request{URL: urlx.ParseOrPanic("/?consistency=strong")}))
	assert.Equal(t, ConsistencyLevelEventual, ConsistencyLevelFromRequest(&http.Request{URL: urlx.ParseOrPanic("/?consistency=eventual")}))
	assert.Equal(t, ConsistencyLevelStrong, ConsistencyLevelFromRequest(&http.Request{URL: urlx.ParseOrPanic("/?consistency=asdf")}))

}

func TestSetTransactionConsistency(t *testing.T) {
	t.Fatalf("todo")
}
