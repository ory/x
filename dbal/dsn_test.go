package dbal

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsSqlite(t *testing.T) {
	require.True(t, IsMemorySQLite(SQLiteInMemory))
	require.True(t, IsMemorySQLite(SQLiteSharedInMemory))
	require.True(t, IsMemorySQLite("sqlite://file:uniquedb:?_fk=true&mode=memory"))
	require.True(t, IsMemorySQLite("sqlite://file:uniquedb:?_fk=true&cache=shared"))
	require.True(t, IsMemorySQLite("sqlite://file:uniquedb:?_fk=true&mode=memory&cache=shared"))
	require.True(t, IsMemorySQLite("sqlite://file:uniquedb:?_fk=true&cache=shared&mode=memory"))
	require.False(t, IsMemorySQLite("sqlite://file:::uniquedb:?_fk=true&mode=memory"))
	require.False(t, IsMemorySQLite("sqlite://"))
	require.False(t, IsMemorySQLite("sqlite://file"))
	require.False(t, IsMemorySQLite("sqlite://file:::"))
	require.False(t, IsMemorySQLite("sqlite://?_fk=true&mode=memory"))
	require.False(t, IsMemorySQLite("sqlite://?_fk=true&cache=shared"))
	require.False(t, IsMemorySQLite("sqlite://file::?_fk=true"))
	require.False(t, IsMemorySQLite("sqlite://file:::?_fk=true"))
	require.False(t, IsMemorySQLite("postgresql://username:secret@localhost:5432/database"))
}
