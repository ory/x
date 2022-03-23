package cloudx

import (
	"fmt"
	"net/url"

	"github.com/ory/x/stringsx"

	"github.com/spf13/cobra"

	"github.com/ory/x/flagx"
)

func NewTunnelCommand(self string, version string) *cobra.Command {
	proxyCmd := &cobra.Command{
		Use:   "tunnel application-url [tunnel-url]",
		Short: fmt.Sprintf("Tunnel Ory on a subdomain of your app or a seperate port your app's domain"),
		Args:  cobra.RangeArgs(1, 2),
		Long: fmt.Sprintf(`This command starts an HTTP tunnel to Ory on the domain `+"`"+`[tunnel-url]`+"`"+` you specify. This
allows to co-locate Ory's APIs and your application on the same top-level-domain,
which is required to get cookies and other security features working.

This tunnel works both in development and in production, for example when deploying a
React, NodeJS, Java, PHP, ... app to a server / the cloud or when developing it locally
on your machine. It is similar to the `+"`"+`%[1]s proxy`+"`"+` command, but it does not require to
route all your application's traffic through it!

Before you start, you need to have a running instance of Ory. Set the environment variable
ORY_SDK_URL to the path where Ory is available. For Ory Cloud, this is the "SDK URL"
which can be found in the "API & Services" section of your Ory Cloud Console.

	$ export ORY_SDK_URL=https://playground.projects.oryapis.com

The first argument `+"`"+`application-url`+"`"+` points to the location of your application. This location
will be used as the default redirect URL for the tunnel, for example after a successful login.

    $ %[1]s tunnel https://www.example.org

It has the same behavior as `+"`"+`%[1]s proxy --default-redirect-url https://example.org/...`+"`"+`.

You can change this behavior using the `+"`"+`--default-redirect-url`+"`"+` flag:

    $ %[1]s tunnel --default-redirect-url /welcome \
		https://www.example.org

The second argument `+"`"+`[tunnel-url]`+"`"+` is optional. It refers to the public URL of this tunnel
(e.g. https://auth.example.org).

If `+"`"+`[tunnel-url]`+"`"+` is not set, it will default to the default host and port the tunnel listens on:

	http://localhost:4000

You must set the `+"`"+`[tunnel-url]`+"`"+` if you are not using the tunnel in local development:

	$ %[1]s tunnel \
		https://www.example.org \
		https://auth.example.org

Please note that you can not set a path in the `+"`"+`[tunnel-url]`+"`"+`!

Per default, the tunnel listens on port 4000. If you want to listen on another port, use the
port flag:

	$ %[1]s tunnel --port 8080 \
		https://www.example.org

If your application URL is available on a non-standard HTTP/HTTPS port, you can set that port in the `+"`"+`application-url`+"`"+`:

	$ %[1]s tunnel \
		https://example.org:1234

If this tunnel runs on a subdomain, and you want Ory's cookies (e.g. the session cookie) to
be available on all of your domain, you can use the following CLI flag to customize the cookie
domain:

	$ %[1]s tunnel \
		--cookie-domain example.org \
		https://www.example.org \
		https://auth.example.org

In contrast to the `+"`"+`%[1]s proxy`+"`"+`, the tunnel does not alter the HTTP headers arriving at your
application and it does not generate any JSON Web Tokens.`, self),

		RunE: func(cmd *cobra.Command, args []string) error {
			port := flagx.MustGetInt(cmd, PortFlag)
			selfURLString := fmt.Sprintf("http://localhost:%d", port)
			if len(args) == 2 {
				selfURLString = args[1]
			}

			selfURL, err := url.ParseRequestURI(selfURLString)
			if err != nil {
				return err
			}

			redirectURL, err := url.Parse(stringsx.Coalesce(flagx.MustGetString(cmd, DefaultRedirectURLFlag), args[0]))
			if err != nil {
				return err
			}

			oryURL, err := getEndpointURL(cmd)
			if err != nil {
				return err
			}

			conf := &config{
				port:              flagx.MustGetInt(cmd, PortFlag),
				noJWT:             true,
				noOpen:            true,
				upstream:          oryURL.String(),
				cookieDomain:      flagx.MustGetString(cmd, CookieDomainFlag),
				publicURL:         selfURL,
				oryURL:            oryURL,
				pathPrefix:        "",
				isTunnel:          true,
				defaultRedirectTo: redirectURL,
			}

			return run(cmd, conf, version, "cloud")
		},
	}

	proxyCmd.Flags().String(CookieDomainFlag, "", "Set a dedicated cookie domain.")
	proxyCmd.Flags().String(ServiceURL, "", "Set the Ory SDK URL.")
	proxyCmd.Flags().Int(PortFlag, portFromEnv(), "The port the proxy should listen on.")
	proxyCmd.Flags().String(DefaultRedirectURLFlag, "", "Set the URL to redirect to per default after e.g. login or account creation.")
	return proxyCmd
}
