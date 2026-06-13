package mysqlutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

type Config struct {
	User     string
	Password string
	Database string
	Host     string
	Port     int
}

func Open(ctx context.Context, params Config) (*sql.DB, error) {
	if strings.TrimSpace(params.User) == "" {
		return nil, errors.New("mysql: user is required")
	}
	if strings.TrimSpace(params.Database) == "" {
		return nil, errors.New("mysql: database is required")
	}
	host := params.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := params.Port
	if port <= 0 {
		port = 3306
	}

	cfg := mysql.Config{
		User:                 params.User,
		Passwd:               params.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", host, port),
		ParseTime:            true,
		AllowNativePasswords: true,
	}
	adminDB, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+quoteIdentifier(params.Database)); err != nil {
		_ = adminDB.Close()
		return nil, fmt.Errorf("mysql: create database %q: %w", params.Database, err)
	}
	if err := adminDB.Close(); err != nil {
		return nil, err
	}

	cfg.DBName = params.Database
	return sql.Open("mysql", cfg.FormatDSN())
}

func quoteIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}
