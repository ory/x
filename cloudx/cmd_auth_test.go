package cloudx

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cloud "github.com/ory/client-go"
	"github.com/ory/x/pointerx"
)

func TestAuthenticator(t *testing.T) {
	configDir := newConfigDir(t)

	t.Run("errors without config and --quiet flag", func(t *testing.T) {
		cmd := NewRootCommand("", "")
		cmd.SetArgs([]string{"auth", "--" + configFlag, configDir, "--quiet"})
		require.Error(t, cmd.Execute())
	})

	password := fakePassword()
	exec := configPasswordAwareCmd(configDir, password)

	signIn := func(t *testing.T, email string) (string, string, error) {
		clearConfig(t, configDir)
		var r bytes.Buffer

		_, _ = r.WriteString("y\n")        // Do you already have an Ory Console account you wish to use? [y/n]: y
		_, _ = r.WriteString(email + "\n") // Email: fakeEmail()

		return exec.Exec(&r, "auth")
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

		stdout, stderr, err := exec.Exec(&r, "auth")
		require.NoError(t, err)

		assert.Contains(t, stderr, "You are now signed in as: "+email, "Expected to be signed in but response was:\n\t%s\n\tstderr: %s", stdout, stderr)
		assert.Contains(t, stdout, email)
		assertConfig(t, configDir, email, name, false)
		clearConfig(t, configDir)

		expectSignInSuccess := func(t *testing.T) {
			stdout, _, err := signIn(t, email)
			require.NoError(t, err)

			assert.Contains(t, stderr, "You are now signed in as: ", email, stdout)
			assertConfig(t, configDir, email, name, false)
		}

		t.Run("sign in with valid data", func(t *testing.T) {
			expectSignInSuccess(t)
		})

		t.Run("forced to reauthenticate on session expiration", func(t *testing.T) {
			cmd := configAwareCmd(configDir)
			expectSignInSuccess(t)
			changeAccessToken(t, configDir)
			var r bytes.Buffer
			r.WriteString("n\n") // Your CLI session has expired. Do you wish to login again as <email>?
			_, stderr, err := cmd.ExecDebug(t, &r, "list", "projects")
			require.Error(t, err)
			assert.Contains(t, stderr, "Your CLI session has expired. Do you wish to log in again as")
		})

		t.Run("user is able to reauthenticate on session expiration", func(t *testing.T) {
			cmd := configAwareCmd(configDir)
			expectSignInSuccess(t)
			changeAccessToken(t, configDir)
			var r bytes.Buffer
			r.WriteString("y\n") // Your CLI session has expired. Do you wish to login again as <email>?
			_, stderr, err := cmd.ExecDebug(t, &r, "list", "projects")
			require.Error(t, err)
			assert.Contains(t, stderr, "Your CLI session has expired. Do you wish to log in again as")
			expectSignInSuccess(t)
		})

		t.Run("expired session with quiet flag returns error", func(t *testing.T) {
			cmd := configAwareCmd(configDir)
			expectSignInSuccess(t)
			changeAccessToken(t, configDir)
			_, stderr, err := cmd.ExecDebug(t, nil, "list", "projects", "-q")
			require.Error(t, err)
			assert.Equal(t, "Your session has expired and you cannot reauthenticate when the --quiet flag is set", err.Error())
			assert.NotContains(t, stderr, "Your CLI session has expired. Do you wish to log in again as")
		})

		t.Run("set up 2fa", func(t *testing.T) {
			expectSignInSuccess(t)
			ac := readConfig(t, configDir)

			c, err := newKratosClient()
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

			_, _, err = c.V0alpha2Api.SubmitSelfServiceSettingsFlow(context.Background()).XSessionToken(ac.SessionToken).Flow(flow.Id).SubmitSelfServiceSettingsFlowBody(cloud.SubmitSelfServiceSettingsFlowBody{
				SubmitSelfServiceSettingsFlowWithTotpMethodBody: &cloud.SubmitSelfServiceSettingsFlowWithTotpMethodBody{
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

				stdout, stderr, err := exec.Exec(&r, "auth")
				require.Error(t, err, stdout)

				assert.Contains(t, stderr, "Please complete the second authentication challenge", stdout)
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

				stdout, stderr, err := exec.Exec(&r, "auth")
				require.NoError(t, err, stdout)

				assert.Contains(t, stderr, "Please complete the second authentication challenge", stdout)
				assert.Contains(t, stderr, "You are now signed in as: ", email, stdout)
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

		stdout, stderr, err := exec.Exec(&r, "auth", "--"+configFlag, configDir)
		require.NoError(t, err)

		assert.Contains(t, stderr, "Your account creation attempt failed. Please try again!", stdout) // First try fails
		assert.Contains(t, stderr, "You are now signed in as: "+email, stdout)                        // Second try succeeds
		assertConfig(t, configDir, email, name, true)
	})

	t.Run("sign in with invalid data", func(t *testing.T) {
		stdout, stderr, err := signIn(t, fakeEmail())
		require.Error(t, err, stdout)

		assert.Contains(t, stderr, "The provided credentials are invalid", stdout)
	})
}
