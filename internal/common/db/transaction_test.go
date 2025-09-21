package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
)

func TestTxManager_RunInTx_Success(t *testing.T) {
	database, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	mock.ExpectBegin()
	mock.ExpectCommit()

	manager := db.NewTxManager(database)

	err = manager.RunInTx(context.Background(), func(ctx context.Context) error {
		tx, ok := db.GetTx(ctx)
		require.True(t, ok)
		require.NotNil(t, tx)
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTxManager_RunInTx_RollbackOnError(t *testing.T) {
	database, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	mock.ExpectBegin()
	mock.ExpectRollback()

	manager := db.NewTxManager(database)
	sentinel := errors.New("boom")

	err = manager.RunInTx(context.Background(), func(ctx context.Context) error {
		_, ok := db.GetTx(ctx)
		require.True(t, ok)
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTxManager_RunInTx_NestedUsesSameTransaction(t *testing.T) {
	database, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	mock.ExpectBegin()
	mock.ExpectCommit()

	manager := db.NewTxManager(database)

	err = manager.RunInTx(context.Background(), func(ctx context.Context) error {
		outerTx, ok := db.GetTx(ctx)
		require.True(t, ok)

		return manager.RunInTx(ctx, func(innerCtx context.Context) error {
			innerTx, innerOK := db.GetTx(innerCtx)
			require.True(t, innerOK)
			require.Same(t, outerTx, innerTx)
			return nil
		})
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// No dedicated tests for mock TxManager; see mock package.
