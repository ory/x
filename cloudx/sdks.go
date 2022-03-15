package cloudx

import (
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/ory/x/stringsx"

	cloud "github.com/ory/client-go"
)

func newKratosClient() (*cloud.APIClient, error) {
	u, err := url.ParseRequestURI(stringsx.Coalesce(os.Getenv("ORY_CLOUD_CONSOLE_URL"), "https://project.console.ory.sh"))
	if err != nil {
		return nil, errors.Wrap(err, "unable to determine the Ory Cloud Project URL")
	}

	conf := cloud.NewConfiguration()
	conf.Servers = cloud.ServerConfigurations{{URL: u.String()}}
	conf.HTTPClient = &http.Client{Timeout: time.Second * 10}

	return cloud.NewAPIClient(conf), nil
}

func newCloudClient(token string) (*cloud.APIClient, error) {
	u, err := url.ParseRequestURI(stringsx.Coalesce(os.Getenv("ORY_CLOUD_API_URL"), "https://api.console.ory.sh"))
	if err != nil {
		return nil, errors.Wrap(err, "unable to determine the Ory Cloud API URL")
	}

	conf := cloud.NewConfiguration()
	conf.Servers = cloud.ServerConfigurations{{URL: u.String()}}
	conf.HTTPClient = NewCloudHTTPClient(token)

	return cloud.NewAPIClient(conf), nil
}
