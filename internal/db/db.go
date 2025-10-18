package db

import (
	"context"
	"database/sql"
	"path/filepath"

	sqlc "github.com/OptimusePrime/petagpt/internal/sqlc"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

const SQLITE_VERSION = 0

var MainDB *sql.DB

func InitDatabase(ctx context.Context) error {
	configDir := viper.GetString("data_dir")

	dbPath := filepath.Join(configDir, "storage.sqlite")
	
	var err error
	MainDB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	var dbVersion int
	row := MainDB.QueryRow("PRAGMA user_version")
	err = row.Scan(&dbVersion)
	if err != nil {
		return err
	}

	if dbVersion == SQLITE_VERSION {
		return nil
	}

	_, err = MainDB.ExecContext(ctx, sqlc.DDL)
	if err != nil {
		return err
	}

	return nil
}
