package networkx

import (
	"context"
	"github.com/gobuffalo/pop/v5"
	"github.com/ory/x/dbal"
	"github.com/ory/x/logrusx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestManager(t *testing.T) {
	ctx := context.Background()

	c, err := pop.NewConnection(&pop.ConnectionDetails{URL: dbal.SQLiteInMemory})
	require.NoError(t, err)
	require.NoError(t, c.Open())

	l := logrusx.New("", "")
	m := NewManager(c, l, nil)

	require.NoError(t, m.MigrateUp(ctx))

	first, err := m.Determine(ctx)
	require.NoError(t, err)

	assert.NotNil(t, first.ID)

	second, err := m.Determine(ctx)
	require.NoError(t, err)

	assert.EqualValues(t, first.ID, second.ID)
}
