package cache

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const cacheTable = "cache_results"

type Cache struct {
	db      *sql.DB
	enabled bool
}

func New(dbPath string) (*Cache, error) {
	if dbPath == "" {
		return &Cache{enabled: false}, nil
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open cache db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if _, err := db.Exec(fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (text TEXT PRIMARY KEY, data TEXT NOT NULL)",
		cacheTable,
	)); err != nil {
		db.Close()
		return nil, fmt.Errorf("create cache table: %w", err)
	}

	return &Cache{db: db, enabled: true}, nil
}

func (c *Cache) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *Cache) Get(text string) (string, error) {
	if !c.enabled {
		return "", nil
	}

	row := c.db.QueryRow(fmt.Sprintf("SELECT data FROM %s WHERE text = ?", cacheTable), text)
	var data string
	if err := row.Scan(&data); err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("cache read: %w", err)
	}
	return data, nil
}

func (c *Cache) Delete(text string) error {
	if !c.enabled {
		return nil
	}
	_, err := c.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE text = ?", cacheTable), text)
	return err
}

func (c *Cache) Set(text string, jsonData string) error {
	if !c.enabled {
		return nil
	}
	_, err := c.db.Exec(
		fmt.Sprintf("INSERT OR REPLACE INTO %s (text, data) VALUES (?, ?)", cacheTable),
		text, jsonData,
	)
	return err
}
