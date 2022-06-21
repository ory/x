package watcherx

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	// m.Run()
	goleak.VerifyTestMain(m)
}
