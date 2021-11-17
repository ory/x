package cloudx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	kratos "github.com/ory/kratos-client-go"
	"github.com/ory/x/assertx"
	"github.com/ory/x/cmdx"
	"github.com/ory/x/pointerx"
	"github.com/ory/x/randx"
)

func fakeEmail() string {
	return fmt.Sprintf("dev+orycye2eda2f162daf6142dd0.%s@ory.dev", randx.MustString(16, randx.AlphaLowerNum))
}

func fakePassword() string {
	return randx.MustString(16, randx.AlphaLowerNum)
}

func fakeName() string {
	return randx.MustString(16, randx.AlphaLowerNum)
}

func newConfigDir(t *testing.T) string {
	homeDir, err := os.MkdirTemp(os.TempDir(), "cloudx-*")
	require.NoError(t, err)
	return filepath.Join(homeDir, "config.json")
}

func readConfig(t *testing.T, configDir string) *AuthContext {
	f, err := os.Open(configDir)
	require.NoError(t, err)
	var ac AuthContext
	require.NoError(t, json.NewDecoder(f).Decode(&ac))
	return &ac
}

func clearConfig(t *testing.T, configDir string) {
	require.NoError(t, os.RemoveAll(configDir))
}

func assertConfig(t *testing.T, configDir string, email string, name string, newsletter bool) {
	ac := readConfig(t, configDir)
	assert.Equal(t, email, ac.IdentityTraits.Email)
	assert.Equal(t, version, ac.Version)
	assert.NotEmpty(t, ac.SessionToken)

	c, err := newConsoleClient("public")
	require.NoError(t, err)

	res, _, err := c.V0alpha2Api.ToSession(context.Background()).XSessionToken(ac.SessionToken).Execute()
	require.NoError(t, err)

	traits, err := json.Marshal(res.Identity.Traits)
	require.NoError(t, err)

	assertx.EqualAsJSONExcept(t, json.RawMessage(`{
  "email": "`+email+`",
  "name": "`+name+`",
  "consent": {
    "newsletter": `+fmt.Sprintf("%v", newsletter)+`,
    "tos": ""
  }
}`), json.RawMessage(traits), []string{"consent.tos"})
	assert.NotEmpty(t, gjson.GetBytes(traits, "consent.tos").String())
}

func TestAuthenticator(t *testing.T) {
	configDir := newConfigDir(t)

	// Use staging
	require.NoError(t, os.Setenv("ORY_CLOUD_CONSOLE_URL", "https://project.console.staging.ory.dev"))

	t.Run("errors without config and --yes flag", func(t *testing.T) {
		cmd := NewRootCommand("", "")
		cmd.SetArgs([]string{"auth", "--cloud-config", configDir, "--yes"})
		require.Error(t, cmd.Execute())
	})

	password := fakePassword()
	exec := cmdx.CommandExecuter{
		New: func() *cobra.Command {
			return NewRootCommand("", "")
		},
		Ctx: context.WithValue(context.Background(), PasswordReader, passwordReader(func(fd int) ([]byte, error) {
			return []byte(password), nil
		})),
		PersistentArgs: []string{"--cloud-config", configDir},
	}

	t.Run("success", func(t *testing.T) {
		email := fakeEmail()
		name := fakeName()

		// Create the account
		var r bytes.Buffer
		_, _ = r.WriteString("n\n")        // Do you already have an Ory Console account you wish to use? [y/n]: n
		_, _ = r.WriteString(email + "\n") // Email: fakeEmail()
		_, _ = r.WriteString(name + "\n")  // Name: fakeName()
		_, _ = r.WriteString("n\n")        // Please inform me about platform and security updates? [y/n]: n
		_, _ = r.WriteString("n\n")        // I accept the Terms of Service [y/n]: n
		_, _ = r.WriteString("y\n")        // I accept the Terms of Service [y/n]: y

		stdout, _, err := exec.Exec(&r, "auth")
		require.NoError(t, err)

		assert.Contains(t, stdout, "You are now signed in as: "+email, stdout)
		assertConfig(t, configDir, email, name, false)
		clearConfig(t, configDir)

		t.Run("sign in with valid data", func(t *testing.T) {
			clearConfig(t, configDir)
			var r bytes.Buffer

			_, _ = r.WriteString("y\n")        // Do you already have an Ory Console account you wish to use? [y/n]: y
			_, _ = r.WriteString(email + "\n") // Email: fakeEmail()

			stdout, _, err := exec.Exec(&r, "auth")
			require.NoError(t, err)

			assert.Contains(t, stdout, "You are now signed in as: ", email, stdout)
			assertConfig(t, configDir, email, name, false)
		})

		t.Run("set up 2fa", func(t *testing.T) {
			clearConfig(t, configDir)
			var r bytes.Buffer

			_, _ = r.WriteString("y\n")        // Do you already have an Ory Console account you wish to use? [y/n]: y
			_, _ = r.WriteString(email + "\n") // Email: fakeEmail()

			stdout, _, err := exec.Exec(&r, "auth")
			require.NoError(t, err, stdout)

			assert.Contains(t, stdout, "You are now signed in as: ", email, stdout)
			assertConfig(t, configDir, email, name, false)

			f, err := os.Open(configDir)
			require.NoError(t, err)
			var ac AuthContext
			require.NoError(t, json.NewDecoder(f).Decode(&ac))

			c, err := newConsoleClient("public")
			require.NoError(t, err)

			flow, _, err := c.V0alpha2Api.InitializeSelfServiceSettingsFlowWithoutBrowser(context.Background()).XSessionToken(ac.SessionToken).Execute()
			require.NoError(t, err)

			var secret string
			for _, node := range flow.Ui.Nodes {
				if node.Type != "text" {
					continue
				}

				attrs := node.Attributes.UiNodeTextAttributes
				if attrs.Text.Id == 1050006 {
					secret = attrs.Text.Text
				}
			}

			require.NotEmpty(t, secret)
			code, err := totp.GenerateCode(secret, time.Now())
			require.NoError(t, err)

			_, _, err = c.V0alpha2Api.SubmitSelfServiceSettingsFlow(context.Background()).XSessionToken(ac.SessionToken).Flow(flow.Id).SubmitSelfServiceSettingsFlowBody(kratos.SubmitSelfServiceSettingsFlowBody{
				SubmitSelfServiceSettingsFlowWithTotpMethodBody: &kratos.SubmitSelfServiceSettingsFlowWithTotpMethodBody{
					TotpCode: pointerx.String(code),
					Method:   "totp",
				},
			}).Execute()
			require.NoError(t, err)
			clearConfig(t, configDir)

			t.Run("sign in fails because second factor is missing", func(t *testing.T) {
				clearConfig(t, configDir)

				var r bytes.Buffer

				_, _ = r.WriteString("y\n")        // Do you already have an Ory Console account you wish to use? [y/n]: y
				_, _ = r.WriteString(email + "\n") // Email: fakeEmail()

				stdout, _, err := exec.Exec(&r, "auth")
				require.Error(t, err, stdout)

				assert.Contains(t, stdout, "Please complete the second authentication challenge", stdout)
				_, err = os.Stat(configDir)
				assert.ErrorIs(t, err, os.ErrNotExist)
			})

			t.Run("sign in succeeds with second factor", func(t *testing.T) {
				clearConfig(t, configDir)

				var r bytes.Buffer

				code, err := totp.GenerateCode(secret, time.Now())
				require.NoError(t, err)
				_, _ = r.WriteString("y\n")        // Do you already have an Ory Console account you wish to use? [y/n]: y
				_, _ = r.WriteString(email + "\n") // Email: fakeEmail()
				_, _ = r.WriteString(code + "\n")  // TOTP code

				stdout, _, err := exec.Exec(&r, "auth")
				require.NoError(t, err, stdout)

				assert.Contains(t, stdout, "Please complete the second authentication challenge", stdout)
				assert.Contains(t, stdout, "You are now signed in as: ", email, stdout)
				assertConfig(t, configDir, email, name, false)
			})
		})
	})

	t.Run("retry sign up on invalid data", func(t *testing.T) {
		clearConfig(t, configDir)

		var r bytes.Buffer

		_, _ = r.WriteString("n\n")                 // Do you already have an Ory Console account you wish to use? [y/n]: n
		_, _ = r.WriteString("not-an-email" + "\n") // Email: fakeEmail()
		_, _ = r.WriteString(fakeName() + "\n")     // Name: fakeName()
		_, _ = r.WriteString("n\n")                 // Please inform me about platform and security updates? [y/n]: n
		_, _ = r.WriteString("y\n")                 // I accept the Terms of Service [y/n]: y

		// Redo the flow
		email := fakeEmail()
		name := fakeName()
		_, _ = r.WriteString(email + "\n") // Email: fakeEmail()
		_, _ = r.WriteString(name + "\n")  // Name: fakeName()
		_, _ = r.WriteString("y\n")        // Please inform me about platform and security updates? [y/n]: n
		_, _ = r.WriteString("y\n")        // I accept the Terms of Service [y/n]: y

		stdout, _, err := exec.Exec(&r, "auth", "--cloud-config", configDir)
		require.NoError(t, err)

		assert.Contains(t, stdout, "Your account creation attempt failed. Please try again!", stdout) // First try fails
		assert.Contains(t, stdout, "You are now signed in as: "+email, stdout)                        // Second try succeeds
		assertConfig(t, configDir, email, name, true)
	})

	t.Run("sign in with invalid data", func(t *testing.T) {
		clearConfig(t, configDir)

		var r bytes.Buffer
		_, _ = r.WriteString("y\n")                           // Do you already have an Ory Console account you wish to use? [y/n]: y
		_, _ = r.WriteString("i-do-not-exist@ory.dev" + "\n") // Email: fakeEmail()

		stdout, _, err := exec.Exec(&r, "auth")
		require.Error(t, err, stdout)

		assert.Contains(t, stdout, "The provided credentials are invalid", stdout)
	})
}
