package cloudx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/ory/x/assertx"
	"github.com/ory/x/cmdx"
	"github.com/ory/x/randx"
)

const testProjectPattern = "ory-cy-e2e-da2f162d-af61-42dd-90dc-e3fcfa7c84a0-"
const testAccountPrefix = "dev+orycye2eda2f162daf6142dd"

func testProjectName() string {
	return testProjectPattern + randx.MustString(16, randx.AlphaLowerNum)
}

func fakeEmail() string {
	return fmt.Sprintf(testAccountPrefix+".%s@ory.dev", randx.MustString(16, randx.AlphaLowerNum))
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
	f, err := os.ReadFile(configDir)
	require.NoError(t, err)
	var ac AuthContext
	require.NoError(t, json.Unmarshal(f, &ac))
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

	c, err := newKratosClient()
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

func configAwareCmd(configDir string) *cmdx.CommandExecuter {
	return &cmdx.CommandExecuter{
		New: func() *cobra.Command {
			return NewRootCommand("", "")
		},
		PersistentArgs: []string{"--" + configFlag, configDir},
	}
}

func configPasswordAwareCmd(configDir, password string) *cmdx.CommandExecuter {
	return &cmdx.CommandExecuter{
		New: func() *cobra.Command {
			return NewRootCommand("", "")
		},
		Ctx: context.WithValue(context.Background(), PasswordReader, passwordReader(func() ([]byte, error) {
			return []byte(password), nil
		})),
		PersistentArgs: []string{"--" + configFlag, configDir},
	}
}

func changeAccessToken(t *testing.T, configDir string) {
	ac := readConfig(t, configDir)
	ac.SessionToken = "12341234"
	data, err := json.Marshal(ac)
	require.NoError(t, err)
	err = os.WriteFile(configDir, data, 0644)
	require.NoError(t, err)

}

func registerAccount(t *testing.T, configDir string) (email, password string) {
	password = fakePassword()
	email = fakeEmail()
	name := fakeName()

	// Create the account
	var r bytes.Buffer
	_, _ = r.WriteString("n\n")        // Do you already have an Ory Console account you wish to use? [y/n]: n
	_, _ = r.WriteString(email + "\n") // Email: fakeEmail()
	_, _ = r.WriteString(name + "\n")  // Name: fakeName()
	_, _ = r.WriteString("n\n")        // Please inform me about platform and security updates? [y/n]: n
	_, _ = r.WriteString("n\n")        // I accept the Terms of Service [y/n]: n
	_, _ = r.WriteString("y\n")        // I accept the Terms of Service [y/n]: y

	exec := cmdx.CommandExecuter{
		New: func() *cobra.Command {
			return NewRootCommand("", "")
		},
		Ctx: context.WithValue(context.Background(), PasswordReader, passwordReader(func() ([]byte, error) {
			return []byte(password), nil
		})),
		PersistentArgs: []string{"--" + configFlag, configDir},
	}

	stdout, stderr, err := exec.Exec(&r, "auth")
	require.NoError(t, err)

	assert.Contains(t, stderr, "You are now signed in as: "+email, stdout)
	assertConfig(t, configDir, email, name, false)
	return email, password
}
