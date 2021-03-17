package dbal

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsSqlite(t *testing.T) {
	require.True(t, IsSqlite(SQLiteInMemory))
	require.True(t, IsSqlite(SQLiteSharedInMemory))
	require.True(t, IsSqlite("sqlite://file:uniquedb:?_fk=true&mode=memory"))
	require.True(t, IsSqlite("sqlite://file:uniquedb:?_fk=true&cache=shared"))
	require.True(t, IsSqlite("sqlite://file:uniquedb:?_fk=true&mode=memory&cache=shared"))
	require.True(t, IsSqlite("sqlite://file:uniquedb:?_fk=true&cache=shared&mode=memory"))
	require.False(t, IsSqlite("sqlite://file:::uniquedb:?_fk=true&mode=memory"))
	require.False(t, IsSqlite("sqlite://"))
	require.False(t, IsSqlite("sqlite://file"))
	require.False(t, IsSqlite("sqlite://file:::"))
	require.False(t, IsSqlite("sqlite://?_fk=true&mode=memory"))
	require.False(t, IsSqlite("sqlite://?_fk=true&cache=shared"))
	require.False(t, IsSqlite("sqlite://file::?_fk=true"))
	require.False(t, IsSqlite("sqlite://file:::?_fk=true"))
	require.False(t, IsSqlite("postgresql://username:secret@localhost:5432/database"))
}
