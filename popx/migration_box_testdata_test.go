package popx_test

import (
	"context"
	"embed"
	"testing"

	"github.com/gobuffalo/pop/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/popx"
)

//go:embed stub/migrations/testdata/*
var testData embed.FS

//go:embed stub/migrations/testdata_migrations/*
var empty embed.FS

//go:embed stub/migrations/check/valid/*
var checkValidFS embed.FS

type testdata struct {
	Data string `db:"data"`
}

func TestMigrationBoxWithTestdata(t *testing.T) {
	c, err := pop.NewConnection(&pop.ConnectionDetails{
		URL: "sqlite://file::memory:?_fk=true",
	})
	require.NoError(t, err)
	require.NoError(t, c.Open())

	mb, err := popx.NewMigrationBox(
		empty,
		popx.NewMigrator(c, logrusx.New("", ""), nil, 0),
		popx.WithTestdata(t, testData))

	require.NoError(t, err)
	assert.Len(t, mb.Migrations["up"], 3)
	assert.Equal(t, "20220513_testdata.sql", mb.Migrations["up"][1].Name)
	assert.Equal(t, "20220514_testdata.sql", mb.Migrations["up"][2].Name)

	require.NoError(t, mb.Up(context.Background()))
	pop.Debug = true
	data := testdata{}
	require.NoError(t, c.First(&data))
	pop.Debug = false
	assert.Equal(t, "testdata", data.Data)
}

func TestMigrationBox_CheckNoErr(t *testing.T) {
	c, err := pop.NewConnection(&pop.ConnectionDetails{
		URL: "sqlite://file::memory:?_fk=true",
	})
	require.NoError(t, err)
	require.NoError(t, c.Open())

	mb, err := popx.NewMigrationBox(
		checkValidFS,
		popx.NewMigrator(c, logrusx.New("", ""), nil, 0),
	)

	require.NoError(t, err)
	assert.Len(t, mb.Migrations["up"], 2)
	assert.Len(t, mb.Migrations["down"], 1)
}
