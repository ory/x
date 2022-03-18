package cloudx

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/imdario/mergo"

	"github.com/ory/x/jsonx"

	"github.com/gofrs/uuid/v3"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tidwall/gjson"
	"golang.org/x/term"

	cloud "github.com/ory/client-go"

	"github.com/ory/x/cmdx"
	"github.com/ory/x/flagx"
	"github.com/ory/x/stringsx"
)

const (
	fileName   = ".ory-cloud.json"
	configFlag = "config"
	osEnvVar   = "ORY_CLOUD_CONFIG_PATH"
	cloudURL   = "ORY_CLOUD_URL"
	version    = "v0alpha0"
	yesFlag    = "yes"
)

func RegisterConfigFlag(f *pflag.FlagSet) {
	f.StringP(configFlag, configFlag[:1], "", "Path to the Ory Cloud configuration file.")
}

func RegisterYesFlag(f *pflag.FlagSet) {
	f.BoolP(yesFlag, yesFlag[:1], false, "Confirm all dialogues with yes.")
}

type AuthContext struct {
	Version         string       `json:"version"`
	SessionToken    string       `json:"session_token"`
	SelectedProject uuid.UUID    `json:"selected_project"`
	IdentityTraits  AuthIdentity `json:"session_identity_traits"`
}

func (i *AuthContext) ID() string {
	return i.IdentityTraits.ID.String()
}

func (*AuthContext) Header() []string {
	return []string{"ID", "EMAIL", "SELECTED_PROJECT"}
}

func (i *AuthContext) Columns() []string {
	return []string{
		i.ID(),
		i.IdentityTraits.Email,
		i.SelectedProject.String(),
	}
}

func (i *AuthContext) Interface() interface{} {
	return i
}

type AuthIdentity struct {
	ID    uuid.UUID
	Email string `json:"email"`
}

type AuthProject struct {
	ID   uuid.UUID `json:"id"`
	Slug string    `json:"slug"`
}

var ErrNoConfig = errors.New("no ory configuration file present")
var ErrNoConfigQuiet = errors.New("please run `ory auth` to initialize your configuration or remove the `--quiet` flag")

func getConfigPath(cmd *cobra.Command) (string, error) {
	path, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrapf(err, "unable to guess your home directory")
	}

	return stringsx.Coalesce(
		os.Getenv(osEnvVar),
		flagx.MustGetString(cmd, configFlag),
		filepath.Join(path, fileName),
	), nil
}

type SnakeCharmer struct {
	ctx              context.Context
	verboseWriter    io.Writer
	verboseErrWriter io.Writer
	configLocation   string
	noConfirm        bool
	isQuiet          bool
	apiDomain        *url.URL
	stdin            *bufio.Reader
	pwReader         passwordReader
}

const PasswordReader = "password_reader"

// NewSnakeCharmer creates a new SnakeCharmer instance which handles cobra CLI commands.
func NewSnakeCharmer(cmd *cobra.Command) (*SnakeCharmer, error) {
	location, err := getConfigPath(cmd)
	if err != nil {
		return nil, err
	}

	var out = cmd.OutOrStdout()
	if flagx.MustGetBool(cmd, cmdx.FlagQuiet) {
		out = io.Discard
	}

	var outErr = cmd.ErrOrStderr()
	if flagx.MustGetBool(cmd, cmdx.FlagQuiet) {
		outErr = io.Discard
	}

	toParse := stringsx.Coalesce(
		os.Getenv(cloudURL),
		"https://project.console.ory.sh",
	)

	apiDomain, err := url.Parse(toParse)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid API endpoint provided: %s", toParse)
	}

	pwReader := func() ([]byte, error) {
		return term.ReadPassword(int(syscall.Stdin))
	}
	if p, ok := cmd.Context().Value(PasswordReader).(passwordReader); ok {
		pwReader = p
	}

	return &SnakeCharmer{
		configLocation:   location,
		noConfirm:        flagx.MustGetBool(cmd, yesFlag),
		isQuiet:          flagx.MustGetBool(cmd, cmdx.FlagQuiet),
		verboseWriter:    out,
		verboseErrWriter: outErr,
		stdin:            bufio.NewReader(cmd.InOrStdin()),
		apiDomain:        apiDomain,
		ctx:              cmd.Context(),
		pwReader:         pwReader,
	}, nil
}

func (h *SnakeCharmer) Stdin() *bufio.Reader {
	return h.stdin
}

func (h *SnakeCharmer) SetDefaultProject(id string) error {
	conf, err := h.readConfig()
	if err != nil {
		return err
	}

	uid, err := uuid.FromString(id)
	if err != nil {
		return err
	}

	conf.SelectedProject = uid
	return h.WriteConfig(conf)
}

func (h *SnakeCharmer) WriteConfig(c *AuthContext) error {
	c.Version = version
	file, err := os.OpenFile(h.configLocation, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return errors.Wrapf(err, "unable to open file for writing at location: %s", file.Name())
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(c); err != nil {
		return errors.Wrapf(err, "unable to write configuration to file: %s", h.configLocation)
	}

	return nil
}

func (h *SnakeCharmer) readConfig() (*AuthContext, error) {
	contents, err := os.ReadFile(h.configLocation)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return new(AuthContext), ErrNoConfig
		}
		return nil, errors.Wrapf(err, "unable to open ory config file location: %s", h.configLocation)
	}

	var c AuthContext
	if err := json.Unmarshal(contents, &c); err != nil {
		return nil, errors.Wrapf(err, "unable to JSON decode the ory config file: %s", h.configLocation)
	}

	return &c, nil
}

func (h *SnakeCharmer) EnsureContext() (*AuthContext, error) {
	c, err := h.readConfig()
	if err != nil {
		if errors.Is(err, ErrNoConfig) {
			if h.isQuiet {
				return nil, ErrNoConfigQuiet
			}
			// Continue to sign in
		} else {
			return nil, err
		}
	}

	if len(c.SessionToken) > 0 {
		_, _ = fmt.Fprintf(h.verboseErrWriter, "You are authenticated as: %s\n", c.IdentityTraits.Email)
		return c, nil
	}

	c, err = h.Authenticate()
	if err != nil {
		return nil, err
	}

	if len(c.SessionToken) == 0 {
		return nil, errors.Errorf("unable to authenticate")
	}

	return c, nil
}

func (h *SnakeCharmer) getField(i interface{}, path string) (*gjson.Result, error) {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(i); err != nil {
		return nil, err
	}
	result := gjson.GetBytes(b.Bytes(), path)
	return &result, nil
}

func (h *SnakeCharmer) signup(c *cloud.APIClient) (*AuthContext, error) {
	flow, _, err := c.V0alpha2Api.InitializeSelfServiceRegistrationFlowWithoutBrowser(h.ctx).Execute()
	if err != nil {
		return nil, err
	}

	var isRetry bool
retryRegistration:
	if isRetry {
		_, _ = fmt.Fprintf(h.verboseErrWriter, "\nYour account creation attempt failed. Please try again!\n\n")
	}
	isRetry = true

	var form cloud.SubmitSelfServiceRegistrationFlowWithPasswordMethodBody
	if err := renderForm(h.stdin, h.pwReader, h.verboseErrWriter, flow.Ui, "password", &form); err != nil {
		return nil, err
	}

	signup, _, err := c.V0alpha2Api.SubmitSelfServiceRegistrationFlow(h.ctx).
		Flow(flow.Id).SubmitSelfServiceRegistrationFlowBody(cloud.SubmitSelfServiceRegistrationFlowBody{
		SubmitSelfServiceRegistrationFlowWithPasswordMethodBody: &form,
	}).Execute()
	if err != nil {
		if e, ok := err.(*cloud.GenericOpenAPIError); ok {
			switch m := e.Model().(type) {
			case *cloud.SelfServiceRegistrationFlow:
				flow = m
				goto retryRegistration
			case cloud.SelfServiceRegistrationFlow:
				flow = &m
				goto retryRegistration
			}
		}

		return nil, errors.WithStack(err)
	}

	sessionToken := *signup.SessionToken
	sess, _, err := c.V0alpha2Api.ToSession(h.ctx).XSessionToken(sessionToken).Execute()
	if err != nil {
		return nil, err
	}

	return h.sessionToContext(sess, sessionToken)
}

func (h *SnakeCharmer) signin(c *cloud.APIClient, sessionToken string) (*AuthContext, error) {
	req := c.V0alpha2Api.InitializeSelfServiceLoginFlowWithoutBrowser(h.ctx)
	if len(sessionToken) > 0 {
		req = req.XSessionToken(sessionToken).Aal("aal2")
	}

	flow, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	var isRetry bool
retryLogin:
	if isRetry {
		_, _ = fmt.Fprintf(h.verboseErrWriter, "\nYour sign in attempt failed. Please try again!\n\n")
	}
	isRetry = true

	var form interface{} = &cloud.SubmitSelfServiceLoginFlowWithPasswordMethodBody{}
	method := "password"
	if len(sessionToken) > 0 {
		var foundTOTP bool
		var foundLookup bool
		for _, n := range flow.Ui.Nodes {
			if n.Group == "totp" {
				foundTOTP = true
			} else if n.Group == "lookup_secret" {
				foundLookup = true
			}
		}
		if !foundLookup && !foundTOTP {
			return nil, errors.New("only TOTP and lookup secrets are supported for two-step verification in the CLI")
		}

		method = "lookup_secret"
		if foundTOTP {
			form = &cloud.SubmitSelfServiceLoginFlowWithTotpMethodBody{}
			method = "totp"
		}
	}

	if err := renderForm(h.stdin, h.pwReader, h.verboseErrWriter, flow.Ui, method, form); err != nil {
		return nil, err
	}

	var body cloud.SubmitSelfServiceLoginFlowBody
	switch e := form.(type) {
	case *cloud.SubmitSelfServiceLoginFlowWithTotpMethodBody:
		body.SubmitSelfServiceLoginFlowWithTotpMethodBody = e
	case *cloud.SubmitSelfServiceLoginFlowWithPasswordMethodBody:
		body.SubmitSelfServiceLoginFlowWithPasswordMethodBody = e
	default:
		panic("unexpected type")
	}

	login, _, err := c.V0alpha2Api.SubmitSelfServiceLoginFlow(h.ctx).XSessionToken(sessionToken).
		Flow(flow.Id).SubmitSelfServiceLoginFlowBody(body).Execute()
	if err != nil {
		if e, ok := err.(*cloud.GenericOpenAPIError); ok {
			switch m := e.Model().(type) {
			case *cloud.SelfServiceLoginFlow:
				flow = m
				goto retryLogin
			case cloud.SelfServiceLoginFlow:
				flow = &m
				goto retryLogin
			}
		}

		return nil, errors.WithStack(err)
	}

	sessionToken = stringsx.Coalesce(*login.SessionToken, sessionToken)
	sess, _, err := c.V0alpha2Api.ToSession(h.ctx).XSessionToken(sessionToken).Execute()
	if err == nil {
		return h.sessionToContext(sess, sessionToken)
	}

	if e, ok := err.(*cloud.GenericOpenAPIError); ok {
		switch gjson.GetBytes(e.Body(), "error.id").String() {
		case "session_aal2_required":
			return h.signin(c, sessionToken)
		}
	}
	return nil, err
}

func (h *SnakeCharmer) sessionToContext(session *cloud.Session, token string) (*AuthContext, error) {
	email, err := h.getField(session.Identity.Traits, "email")
	if err != nil {
		return nil, err
	}

	return &AuthContext{
		Version:      version,
		SessionToken: token,
		IdentityTraits: AuthIdentity{
			Email: email.String(),
			ID:    uuid.FromStringOrNil(session.Identity.Id),
		},
	}, nil
}

func (h *SnakeCharmer) Authenticate() (*AuthContext, error) {
	if h.isQuiet {
		return nil, errors.New("can not sign in or sign up when flag --quiet is set")
	}

	ac, err := h.readConfig()
	if err != nil {
		if !errors.Is(err, ErrNoConfig) {
			return nil, err
		}
	}

	if len(ac.SessionToken) > 0 {
		if !h.noConfirm {
			ok, err := cmdx.AskScannerForConfirmation(fmt.Sprintf("You are signed in as \"%s\" already. Do you wish to authenticate with another account?", ac.IdentityTraits.Email), h.stdin, h.verboseErrWriter)
			if err != nil {
				return nil, err
			} else if !ok {
				return ac, nil
			}
			_, _ = fmt.Fprintf(h.verboseErrWriter, "Ok, signing you out!\n")
		}

		if err := h.SignOut(); err != nil {
			return nil, err
		}
	}

	c, err := newKratosClient()
	if err != nil {
		return nil, err
	}

	signIn, err := cmdx.AskScannerForConfirmation("Do you already have an Ory Console account you wish to use?", h.stdin, h.verboseErrWriter)
	if err != nil {
		return nil, err
	}

	var retry bool
	if retry {
		_, _ = fmt.Fprintln(h.verboseErrWriter, "Unable to Authenticate you, please try again.")
	}

	if signIn {
		ac, err = h.signin(c, "")
		if err != nil {
			return nil, err
		}
	} else {
		_, _ = fmt.Fprintln(h.verboseErrWriter, "Great to have you here, creating an Ory Cloud account is absolutely free and only requires to answer four easy questions.")

		ac, err = h.signup(c)
		if err != nil {
			return nil, err
		}
	}

	if err := h.WriteConfig(ac); err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(h.verboseErrWriter, "You are now signed in as: %s\n", ac.IdentityTraits.Email)

	return ac, nil
}

func (h *SnakeCharmer) SignOut() error {
	return h.WriteConfig(new(AuthContext))
}

func (h *SnakeCharmer) ListProjects() ([]cloud.Project, error) {
	ac, err := h.EnsureContext()
	if err != nil {
		return nil, err
	}

	c, err := newCloudClient(ac.SessionToken)
	if err != nil {
		return nil, err
	}

	projects, res, err := c.V0alpha2Api.ListProjects(h.ctx).Execute()
	if err != nil {
		return nil, handleError("unable to list projects", res, err)
	}

	return projects, nil
}

func (h *SnakeCharmer) GetProject(id string) (*cloud.Project, error) {
	ac, err := h.EnsureContext()
	if err != nil {
		return nil, err
	}

	c, err := newCloudClient(ac.SessionToken)
	if err != nil {
		return nil, err
	}

	project, res, err := c.V0alpha2Api.GetProject(h.ctx, id).Execute()
	if err != nil {
		return nil, handleError("unable to get project", res, err)
	}

	return project, nil
}

func (h *SnakeCharmer) CreateProject(name string) (*cloud.Project, error) {
	ac, err := h.EnsureContext()
	if err != nil {
		return nil, err
	}

	c, err := newCloudClient(ac.SessionToken)
	if err != nil {
		return nil, err
	}

	project, res, err := c.V0alpha2Api.CreateProject(h.ctx).CreateProjectBody(*cloud.NewCreateProjectBody(strings.TrimSpace(name))).Execute()
	if err != nil {
		return nil, handleError("unable to list projects", res, err)
	}

	if err := h.SetDefaultProject(project.Id); err != nil {
		return nil, err
	}

	return project, nil
}

func handleError(message string, res *http.Response, err error) error {
	if e, ok := err.(*cloud.GenericOpenAPIError); ok {
		return errors.Wrapf(err, "%s: %s", message, e.Body())
	}
	body, _ := ioutil.ReadAll(res.Body)
	return errors.Wrapf(err, "%s: %s", message, body)
}

type patch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

func toPatch(op string, values []string) (patches []cloud.JsonPatch, err error) {
	for _, v := range values {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return nil, errors.Errorf("patches must be in format of `/some/config/key=some-value` but got: %s", v)
		} else if !gjson.Valid(parts[1]) {
			return nil, errors.Errorf("value for %s must be valid JSON but got: %s", parts[0], parts[1])
		}

		config, err := jsonx.EmbedSources(json.RawMessage(parts[1]), jsonx.WithIgnoreKeys("$id", "$schema"), jsonx.WithOnlySchemes("file"))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		var value interface{}
		if err := json.Unmarshal(config, &value); err != nil {
			return nil, errors.WithStack(err)
		}

		patches = append(patches, cloud.JsonPatch{Op: op, Path: parts[0], Value: value})
	}
	return patches, nil
}

func (h *SnakeCharmer) PatchProject(id string, raw []json.RawMessage, add, replace, del []string) (*cloud.SuccessfulProjectUpdate, error) {
	ac, err := h.EnsureContext()
	if err != nil {
		return nil, err
	}

	c, err := newCloudClient(ac.SessionToken)
	if err != nil {
		return nil, err
	}

	var patches []cloud.JsonPatch
	for _, r := range raw {
		config, err := jsonx.EmbedSources(r, jsonx.WithIgnoreKeys("$id", "$schema"), jsonx.WithOnlySchemes("file"))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		var p []cloud.JsonPatch
		if err := json.NewDecoder(bytes.NewReader(config)).Decode(&p); err != nil {
			return nil, errors.WithStack(err)
		}
		patches = append(patches, p...)
	}

	if v, err := toPatch("add", add); err != nil {
		return nil, err
	} else {
		//revive:disable indent-error-flow
		patches = append(patches, v...)
	}

	if v, err := toPatch("replace", replace); err != nil {
		return nil, err
	} else {
		//revive:disable indent-error-flow
		patches = append(patches, v...)
	}

	for _, del := range del {
		patches = append(patches, cloud.JsonPatch{Op: "remove", Path: del})
	}

	res, _, err := c.V0alpha2Api.PatchProject(h.ctx, id).JsonPatch(patches).Execute()
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (h *SnakeCharmer) UpdateProject(id string, name string, configs []json.RawMessage) (*cloud.SuccessfulProjectUpdate, error) {
	ac, err := h.EnsureContext()
	if err != nil {
		return nil, err
	}

	c, err := newCloudClient(ac.SessionToken)
	if err != nil {
		return nil, err
	}

	for k := range configs {
		config, err := jsonx.EmbedSources(
			configs[k],
			jsonx.WithIgnoreKeys(
				"$id",
				"$schema",
			),
			jsonx.WithOnlySchemes(
				"file",
			),
		)
		if err != nil {
			return nil, err
		}
		configs[k] = config
	}

	interim := make(map[string]interface{})
	for _, config := range configs {
		var decoded map[string]interface{}
		if err := json.Unmarshal(config, &decoded); err != nil {
			return nil, errors.WithStack(err)
		}

		if err := mergo.Merge(&interim, decoded, mergo.WithAppendSlice, mergo.WithOverride); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	var payload cloud.UpdateProject
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(interim); err != nil {
		return nil, errors.WithStack(err)
	}
	dec := json.NewDecoder(&b)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return nil, errors.WithStack(err)
	}

	if name != "" {
		payload.Name = name
	} else if payload.Name == "" {
		res, _, err := c.V0alpha2Api.GetProject(h.ctx, id).Execute()
		if err != nil {
			return nil, err
		}
		payload.Name = res.Name
	}

	res, _, err := c.V0alpha2Api.UpdateProject(h.ctx, id).UpdateProject(payload).Execute()
	if err != nil {
		return nil, err
	}
	return res, nil
}
