package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/feedback/migrations"
)

// Migrate 逐版本执行 migrations/ 内嵌的 up SQL，记录到 schema_migrations，幂等。
func Migrate(dsn string) error {
	// golang-migrate 的 mysql driver 只接受 scheme://host/db 形式的 DSN
	src, err := iofs.New(migrations.FS(), ".")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, "mysql://"+dsn)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
