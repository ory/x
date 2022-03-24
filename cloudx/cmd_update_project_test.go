package cloudx

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/ory/x/assertx"
	"github.com/ory/x/cmdx"
	"github.com/ory/x/snapshotx"
)

//go:embed fixtures/update/json/config.json
var fixture []byte

func TestUpdateProject(t *testing.T) {
	configDir := newConfigDir(t)
	cmd := configAwareCmd(configDir)
	email, password := registerAccount(t, configDir)

	project := createProject(t, configDir)
	t.Run("is able to update a project", func(t *testing.T) {
		stdout, _, err := cmd.ExecDebug(t, nil, "update", "project", project, "--format", "json", "--file", "./fixtures/update/json/config.json")
		require.NoError(t, err)

		assertx.EqualAsJSONExcept(t, json.RawMessage(fixture), json.RawMessage(stdout), []string{
			"id",
			"revision_id",
			"state",
			"slug",
			"services.identity.config.serve",
			"services.identity.config.cookies",
			"services.identity.config.identity.default_schema_id",
			"services.identity.config.identity.schemas",
			"services.identity.config.session.cookie",
		})

		snapshotx.SnapshotTExcept(t, json.RawMessage(stdout), []string{
			"id",
			"revision_id",
			"slug",
			"services.identity.config.serve.public.base_url",
			"services.identity.config.serve.admin.base_url",
			"services.identity.config.session.cookie.domain",
			"services.identity.config.session.cookie.name",
			"services.identity.config.cookies.domain",
		})
	})
	t.Run("is able to update a projects name", func(t *testing.T) {
		name := fakeName()
		stdout, _, err := cmd.ExecDebug(t, nil, "update", "project", project, "--name", name, "--format", "json", "--file", "./fixtures/update/json/config.json")
		require.NoError(t, err)
		assert.Equal(t, name, gjson.Get(stdout, "name").String())
	})

	t.Run("prints good error messages for failing schemas", func(t *testing.T) {
		updatedName := testProjectName()
		stdout, stderr, err := cmd.ExecDebug(t, nil, "update", "project", project, "--name", updatedName, "--format", "json", "--file", "./fixtures/update/fail/config.json")
		require.ErrorIs(t, err, cmdx.ErrNoPrintButFail)

		t.Run("stdout", func(t *testing.T) {
			snapshotx.SnapshotTExcept(t, stdout, nil)
		})
		t.Run("stderr", func(t *testing.T) {
			assert.Contains(t, stderr, "oneOf failed")
		})
	})

	t.Run("is able to update a project after authenticating", func(t *testing.T) {
		configDir := newConfigDir(t)
		cmd := configPasswordAwareCmd(configDir, password)
		// Create the account
		var r bytes.Buffer
		r.WriteString("y\n")        // Do you already have an Ory Console account you wish to use? [y/n]: y
		r.WriteString(email + "\n") // Email fakeEmail()
		_, _, err := cmd.ExecDebug(t, &r, "update", "project", project, "--format", "json", "--file", "./fixtures/update/json/config.json")
		require.NoError(t, err)
	})
}
