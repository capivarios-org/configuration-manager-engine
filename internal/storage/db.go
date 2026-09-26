package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/sijms/go-ora/v2"

	"github.com/capivarios-org/configuration-manager-engine/internal/config"
	"github.com/capivarios-org/configuration-manager-engine/internal/domain"
)

// DBClient gerencia a conexão com o banco de dados ativo (Postgres, MySQL, MariaDB ou Oracle)
type DBClient struct {
	db     *sql.DB
	driver string
}

// NewDBClient estabelece conexão com o banco de dados configurado
func NewDBClient(cfg *config.Config) (*DBClient, error) {
	var sqlDriver string
	var dsn string

	switch cfg.DBDriver {
	case "postgresql", "postgres":
		sqlDriver = "pgx"
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	case "mysql":
		sqlDriver = "mysql"
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	case "mariadb":
		sqlDriver = "mysql"
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	case "oracle":
		sqlDriver = "oracle"
		dsn = fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	default:
		return nil, fmt.Errorf("driver de banco não suportado: %s", cfg.DBDriver)
	}

	db, err := sql.Open(sqlDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir conexão (%s): %w", cfg.DBDriver, err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &DBClient{
		db:     db,
		driver: cfg.DBDriver,
	}, nil
}

// Ping verifica a conectividade com o banco
func (c *DBClient) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Close fecha a conexão
func (c *DBClient) Close() error {
	return c.db.Close()
}

// FetchAllFlags carrega todas as flags ativas do banco compartilhado
func (c *DBClient) FetchAllFlags(ctx context.Context) ([]domain.Flag, error) {
	query := `SELECT flag_key, flag_value, is_enabled, version, updated_at FROM cm_flags`
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []domain.Flag
	for rows.Next() {
		var f domain.Flag
		var rawVal string
		if err := rows.Scan(&f.Key, &rawVal, &f.IsEnabled, &f.Version, &f.UpdatedAt); err != nil {
			return nil, err
		}
		f.Value = []byte(rawVal)
		flags = append(flags, f)
	}
	return flags, rows.Err()
}

// FetchAllFolders carrega todos os agrupamentos (folders) do banco compartilhado
func (c *DBClient) FetchAllFolders(ctx context.Context) ([]domain.Folder, error) {
	query := `SELECT folder_key, flag_key, flag_value, version, updated_at FROM cm_folder_flags`
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	folderMap := make(map[string]*domain.Folder)
	for rows.Next() {
		var folderKey, flagKey, rawVal string
		var version int64
		var updatedAt time.Time

		if err := rows.Scan(&folderKey, &flagKey, &rawVal, &version, &updatedAt); err != nil {
			return nil, err
		}

		folder, exists := folderMap[folderKey]
		if !exists {
			folder = &domain.Folder{
				Key:       folderKey,
				Flags:     make(map[string]json.RawMessage),
				Version:   version,
				UpdatedAt: updatedAt,
			}
			folderMap[folderKey] = folder
		}
		folder.Flags[flagKey] = json.RawMessage(rawVal)
		if version > folder.Version {
			folder.Version = version
		}
		if updatedAt.After(folder.UpdatedAt) {
			folder.UpdatedAt = updatedAt
		}
	}

	result := make([]domain.Folder, 0, len(folderMap))
	for _, f := range folderMap {
		result = append(result, *f)
	}
	return result, rows.Err()
}
