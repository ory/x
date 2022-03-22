package cloudx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ory/x/proxy"

	"github.com/gofrs/uuid/v3"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/square/go-jose/v3"
	"github.com/square/go-jose/v3/jwt"
	"github.com/tidwall/gjson"
	"github.com/urfave/negroni"

	"github.com/ory/graceful"
	"github.com/ory/herodot"
	"github.com/ory/x/httpx"
	"github.com/ory/x/jwksx"
	"github.com/ory/x/logrusx"
	"github.com/ory/x/urlx"
)

const (
	PortFlag               = "port"
	OpenFlag               = "open"
	WithoutJWTFlag         = "no-jwt"
	CookieDomainFlag       = "cookie-domain"
	DefaultRedirectURLFlag = "default-redirect-url"
	ServiceURL             = "sdk-url"
)

type config struct {
	port              int
	noOpen            bool
	noJWT             bool
	upstream          string
	cookieDomain      string
	publicURL         *url.URL
	oryURL            *url.URL
	pathPrefix        string
	defaultRedirectTo *url.URL
	isTunnel          bool
}

func portFromEnv() int {
	var port int64 = 4000
	if p, _ := strconv.ParseInt(os.Getenv("PORT"), 10, 64); p != 0 {
		port = p
	}
	return int(port)
}

func run(cmd *cobra.Command, conf *config, version string, name string) error {
	upstream, err := url.ParseRequestURI(conf.upstream)
	if err != nil {
		return errors.Wrap(err, "unable to parse upstream URL")
	}

	l := logrusx.New("ory/"+strings.ToLower(name), version)

	writer := herodot.NewJSONWriter(l)

	mw := negroni.New()

	signer, key, err := newSigner(l, conf)
	if err != nil {
		return errors.WithStack(err)
	}

	endpoint, err := getEndpointURL(cmd)
	if err != nil {
		return errors.WithStack(err)
	}

	mw.UseFunc(func(w http.ResponseWriter, r *http.Request, n http.HandlerFunc) {
		// Disable HSTS because it is very annoying to use in localhost.
		w.Header().Set("Strict-Transport-Security", "max-age=0;")
		n(w, r)
	})

	mw.UseFunc(checkOry(conf, l, writer, key, signer, endpoint)) // This must be the last method before the handler

	mw.UseHandler(proxy.New(
		func(_ context.Context, r *http.Request) (*proxy.HostConfig, error) {
			if conf.isTunnel || strings.HasPrefix(r.URL.Path, conf.pathPrefix) {
				return &proxy.HostConfig{
					CookieDomain:   conf.cookieDomain,
					UpstreamHost:   conf.oryURL.Host,
					UpstreamScheme: conf.oryURL.Scheme,
					TargetHost:     conf.oryURL.Host,
					PathPrefix:     conf.pathPrefix,
				}, nil
			}

			return &proxy.HostConfig{
				CookieDomain:   conf.cookieDomain,
				UpstreamHost:   upstream.Host,
				UpstreamScheme: upstream.Scheme,
				TargetHost:     upstream.Host,
				PathPrefix:     "",
			}, nil
		},
		proxy.WithRespMiddleware(func(resp *http.Response, config *proxy.HostConfig, body []byte) ([]byte, error) {
			l, err := resp.Location()
			if err == nil {
				// Redirect to main page if path is the default ui welcome page.
				if l.Path == filepath.Join(conf.pathPrefix, "/ui/welcome") {
					resp.Header.Set("Location", conf.defaultRedirectTo.String())
				}
			}

			return body, nil
		}),
		proxy.WithReqMiddleware(func(r *http.Request, c *proxy.HostConfig, body []byte) ([]byte, error) {
			if r.URL.Host == conf.oryURL.Host {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, conf.pathPrefix)
				r.Host = conf.oryURL.Host
			}

			return body, nil
		})))

	cleanup := func() error {
		return nil
	}

	proto := "http"
	addr := fmt.Sprintf(":%d", conf.port)
	server := graceful.WithDefaults(&http.Server{
		Addr:    addr,
		Handler: mw,
	})

	if conf.isTunnel {
		l.Printf("Starting the %s tunnel on: %s", proto, server.Addr)
		l.Printf(`To access Ory %s via the tunnel, open:

	%s`, name, conf.publicURL.String())
	} else {
		l.Printf("Starting the %s reverse proxy on: %s", proto, server.Addr)
		l.Printf(`To access your application via the Ory %s Proxy, open:

	%s`, name, conf.publicURL.String())
	}

	if !conf.noOpen {
		// #nosec G204 - this is ok
		if err := exec.Command("open", conf.publicURL.String()).Run(); err != nil {
			l.WithError(err).Warn("Unable to automatically open the proxy URL in your browser. Please open it manually!")
		}
	}

	if err := graceful.Graceful(func() error {
		return server.ListenAndServe()
	}, func(ctx context.Context) error {
		l.Println("http server was shutdown gracefully")
		if err := server.Shutdown(ctx); err != nil {
			return err
		}

		return cleanup()
	}); err != nil {
		l.Fatalf("Failed to gracefully shutdown %s server because: %s\n", proto, err)
	}

	return nil
}

func newSigner(l *logrusx.Logger, conf *config) (jose.Signer, *jose.JSONWebKeySet, error) {
	if conf.noJWT {
		return nil, &jose.JSONWebKeySet{}, nil
	}

	l.WithField("started_at", time.Now()).Info("")
	key, err := jwksx.GenerateSigningKeys(
		uuid.Must(uuid.NewV4()).String(),
		"ES256",
		0,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to generate JSON Web Key")
	}
	sig, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: key.Keys[0].Key}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create signer")
	}
	l.WithField("completed_at", time.Now()).Info("ES256 JSON Web Key generation completed.")
	return sig, key, nil
}

func checkOry(conf *config, l *logrusx.Logger, writer herodot.Writer, keys *jose.JSONWebKeySet, sig jose.Signer, endpoint *url.URL) func(http.ResponseWriter, *http.Request, http.HandlerFunc) {
	hc := httpx.NewResilientClient(httpx.ResilientClientWithMaxRetry(5), httpx.ResilientClientWithMaxRetryWait(time.Millisecond*5), httpx.ResilientClientWithConnectionTimeout(time.Second*2))

	var publicKeys jose.JSONWebKeySet
	for _, key := range keys.Keys {
		publicKeys.Keys = append(publicKeys.Keys, key.Public())
	}

	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if !conf.noJWT && r.URL.Path == filepath.Join(conf.pathPrefix, "/proxy/jwks.json") {
			writer.Write(w, r, publicKeys)
			return
		}

		switch r.URL.Path {
		case filepath.Join(conf.pathPrefix, "/jwks.json"):
			writer.Write(w, r, publicKeys)
			return
		}

		session, err := checkSession(hc, r, endpoint)
		r.Header.Del("Authorization")
		if err != nil || !gjson.GetBytes(session, "active").Bool() {
			next(w, r)
			return
		}

		if conf.noJWT || (len(conf.pathPrefix) > 0 && strings.HasPrefix(r.URL.Path, conf.pathPrefix)) {
			next(w, r)
			return
		}

		now := time.Now().UTC()
		raw, err := jwt.Signed(sig).Claims(&jwt.Claims{
			Issuer:    endpoint.String(),
			Subject:   gjson.GetBytes(session, "identity.id").String(),
			Expiry:    jwt.NewNumericDate(now.Add(time.Minute)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.Must(uuid.NewV4()).String(),
		}).Claims(map[string]interface{}{"session": session}).CompactSerialize()
		if err != nil {
			writer.WriteError(w, r, err)
			return
		}

		r.Header.Set("Authorization", "Bearer "+raw)
		next(w, r)
	}
}

func checkSession(c *retryablehttp.Client, r *http.Request, target *url.URL) (json.RawMessage, error) {
	target = urlx.Copy(target)
	target.Path = filepath.Join(target.Path, "api", "kratos", "public", "sessions", "whoami")
	req, err := retryablehttp.NewRequest("GET", target.String(), nil)
	if err != nil {
		return nil, errors.WithStack(herodot.ErrInternalServerError)
	}

	req.Header.Set("Cookie", r.Header.Get("Cookie"))
	req.Header.Set("Authorization", r.Header.Get("Authorization"))
	req.Header.Set("X-Session-Token", r.Header.Get("X-Session-Token"))
	req.Header.Set("X-Request-Id", r.Header.Get("X-Request-Id"))
	req.Header.Set("Accept", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return nil, errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Unable to call session checker: %s", err).WithWrap(err))
	}
	defer res.Body.Close()

	var body json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Unable to decode session to JSON: %s", err).WithWrap(err))
	}

	return body, nil
}
