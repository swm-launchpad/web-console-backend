package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type txManager struct {
	db      *sql.DB
	options sql.TxOptions
}

type TxManagerOption func(*sql.TxOptions)

func WithIsolationLevel(level sql.IsolationLevel) TxManagerOption {
	return func(opts *sql.TxOptions) {
		opts.Isolation = level
	}
}

func WithReadOnly() TxManagerOption {
	return func(opts *sql.TxOptions) {
		opts.ReadOnly = true
	}
}

type txContextKey struct{}

func NewTxManager(db *sql.DB, opts ...TxManagerOption) TxManager {
	manager := &txManager{db: db}
	for _, opt := range opts {
		opt(&manager.options)
	}
	return manager
}

func (m *txManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if _, ok := GetTx(ctx); ok {
		return fn(ctx)
	}

	tx, err := m.db.BeginTx(ctx, &m.options)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	txCtx := context.WithValue(ctx, txContextKey{}, tx)

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Join(err, fmt.Errorf("rollback tx: %w", rbErr))
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func GetTx(ctx context.Context) (*sql.Tx, bool) {
	if ctx == nil {
		return nil, false
	}
	tx, ok := ctx.Value(txContextKey{}).(*sql.Tx)
	return tx, ok
}

// WithTx attaches a transaction to the context.
// This allows manually created transactions to be used with repositories
// that expect transactions in the context.
func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}
