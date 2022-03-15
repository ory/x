package cloudx

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestCreateProject(t *testing.T) {
	configDir := newConfigDir(t)
	registerAccount(t, configDir)

	assertResult := func(t *testing.T, configDir string, stdout string, expectedName string) {
		ac := readConfig(t, configDir)
		assert.Equal(t, gjson.Get(stdout, "id").String(), ac.SelectedProject.String())
		assert.NotEmpty(t, gjson.Get(stdout, "slug").String())
		assert.Equal(t, expectedName, gjson.Get(stdout, "name").String())
	}

	t.Run("is able to create a project", func(t *testing.T) {
		cmd := configAwareCmd(configDir)
		name := testProjectName()
		stdout, _, err := cmd.ExecDebug(t, nil, "create", "project", "--name", name, "--format", "json")
		require.NoError(t, err)
		t.Logf("stdout: %s", stdout)
		assertResult(t, configDir, stdout, name)
	})

	t.Run("is able to create a project and use name from stdin", func(t *testing.T) {
		cmd := configAwareCmd(configDir)
		name := testProjectName()
		stdin := bytes.NewBufferString(name + "\n")
		stdout, _, err := cmd.ExecDebug(t, stdin, "create", "project", "--format", "json")
		require.NoError(t, err)
		assertResult(t, configDir, stdout, name)
	})

	t.Run("is not able to create a project if no name flag and quiet flag", func(t *testing.T) {
		cmd := configAwareCmd(configDir)
		name := testProjectName()
		stdin := bytes.NewBufferString(name)
		_, stderr, err := cmd.ExecDebug(t, stdin, "create", "project", "--quiet")
		require.Error(t, err)
		assert.Contains(t, stderr, "you must specify the --name flag when using --quiet")
	})

	t.Run("is not able to create a project if not authenticated and quiet flag", func(t *testing.T) {
		configDir := newConfigDir(t)
		name := testProjectName()
		cmd := configAwareCmd(configDir)
		_, _, err := cmd.ExecDebug(t, nil, "create", "project", "--name", name, "--format", "json", "--quiet")
		require.ErrorIs(t, err, ErrNoConfigQuiet)
	})

	t.Run("is able to create a project after authenticating", func(t *testing.T) {
		configDir := newConfigDir(t)
		name := testProjectName()
		password := fakePassword()
		email := fakeEmail()
		cmd := configPasswordAwareCmd(configDir, password)
		// Create the account
		var r bytes.Buffer
		r.WriteString("n\n")        // Do you already have an Ory Console account you wish to use? [y/n]: n
		r.WriteString(email + "\n") // Email fakeEmail()
		r.WriteString(name + "\n")  // Name: fakeName()
		r.WriteString("n\n")        // Please inform me about platform and security updates? [y/n]: n
		r.WriteString("n\n")        // I accept the Terms of Service [y/n]: n
		r.WriteString("y\n")        // I accept the Terms of Service [y/n]: y
		stdout, stderr, err := cmd.ExecDebug(t, &r, "create", "project", "--name", name, "--format", "json")
		require.NoError(t, err)
		t.Logf("stdout: %s", stdout)
		t.Logf("stderr: %s", stderr)
		assertResult(t, configDir, stdout, name)
	})
}
