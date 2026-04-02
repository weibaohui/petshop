package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetDBState() {
	Close()
	db = nil
	dbInited = false
	dbErr = nil
}

func TestInitDB_Success(t *testing.T) {
	resetDBState()
	t.Cleanup(resetDBState)

	err := InitDB(":memory:")
	require.NoError(t, err)
	assert.NotNil(t, GetDB())
	assert.True(t, dbInited)
}

func TestInitDB_AlreadyInitialized(t *testing.T) {
	resetDBState()
	t.Cleanup(resetDBState)

	err := InitDB(":memory:")
	require.NoError(t, err)

	// Second InitDB should return the cached result (nil error) without re-opening
	err2 := InitDB(":memory:")
	assert.NoError(t, err2)
	assert.Equal(t, GetDB(), db)
}

func TestInitDB_InvalidPath(t *testing.T) {
	resetDBState()
	t.Cleanup(resetDBState)

	// A path in a non-existent directory causes createTables (and thus InitDB) to fail
	err := InitDB("/nonexistent_dir_xyz_for_test/db.sqlite")
	assert.Error(t, err)
	assert.True(t, dbInited)
	assert.NotNil(t, dbErr)
}

func TestGetDB_BeforeInit(t *testing.T) {
	resetDBState()
	t.Cleanup(resetDBState)

	assert.Nil(t, GetDB())
}

func TestClose(t *testing.T) {
	resetDBState()
	t.Cleanup(resetDBState)

	err := InitDB(":memory:")
	require.NoError(t, err)

	err = Close()
	assert.NoError(t, err)
}

func TestClose_NilDB(t *testing.T) {
	resetDBState()
	t.Cleanup(resetDBState)

	err := Close()
	assert.NoError(t, err)
}

func TestCreateTables_Idempotent(t *testing.T) {
	resetDBState()
	t.Cleanup(resetDBState)

	err := InitDB(":memory:")
	require.NoError(t, err)

	// Calling createTables again should succeed because of IF NOT EXISTS
	err = createTables()
	assert.NoError(t, err)
}
