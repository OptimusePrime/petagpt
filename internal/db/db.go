package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	sqlc "github.com/OptimusePrime/petagpt/internal/sqlc"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

var MainDB *sql.DB

func InitDatabase(ctx context.Context) error {
	configDir := filepath.Dir(viper.GetString("data_dir"))

	dbPath := filepath.Join(configDir, "storage.sqlite")

	var err error
	MainDB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	_, err = os.Stat(dbPath)
	if !os.IsNotExist(err) {
		return nil
	}

	_, err = MainDB.ExecContext(ctx, sqlc.DDL)
	if err != nil {
		return err
	}

	return nil
}
