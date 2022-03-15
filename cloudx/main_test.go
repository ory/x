package cloudx

import (
	"os"
	"testing"

	"github.com/ory/x/stringsx"
)

func TestMain(m *testing.M) {
	// Run the tests
	result := m.Run()

	// Use staging
	if err := os.Setenv("ORY_CLOUD_CONSOLE_URL",
		stringsx.Coalesce(os.Getenv("ORY_CLOUD_CONSOLE_URL"),
			"https://project.console.staging.ory.dev")); err != nil {
		panic(err)
	}

	// Exit appropriately
	os.Exit(result)
}
