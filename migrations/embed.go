// Package migrations — SQL-миграции goose, вшитые в бинарь (применяются при старте, если postgres.auto_migrate).
package migrations

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

//go:embed postgres/*.sql
var FS embed.FS

func Up(ctx context.Context, pool *pgxpool.Pool, log *zap.Logger) error {
	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	goose.SetBaseFS(FS)
	goose.SetLogger(gooseLogger{log.Named("goose").Sugar()})
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.UpContext(ctx, db, "postgres"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

type gooseLogger struct{ *zap.SugaredLogger }

func (l gooseLogger) Printf(format string, v ...any) {
	l.Infof(strings.TrimSpace(format), v...)
}
