package cloudx

import (
	"net/url"
	"os"

	"github.com/pkg/errors"

	"github.com/ory/x/stringsx"

	kratos "github.com/ory/kratos-client-go"
)

func newConsoleClient(port string) (*kratos.APIClient, error) {
	u, err := url.ParseRequestURI(stringsx.Coalesce(os.Getenv("ORY_CLOUD_CONSOLE_URL"), "https://project.console.ory.sh/"))
	if err != nil {
		return nil, errors.Wrap(err, "unable to determine the Ory Cloud Project URL")
	}

	u.Path = "/api/kratos/" + port
	conf := kratos.NewConfiguration()
	conf.Servers = kratos.ServerConfigurations{{URL: u.String()}}
	conf.HTTPClient = NewHTTPClient()

	return kratos.NewAPIClient(conf), nil
}
