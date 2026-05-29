package dict

import (
	"bytes"
	"compress/zlib"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// OfflineDictionary provides offline word lookup from pre-built SQLite databases.
// Each database corresponds to one language pair and is read-only at runtime.
type OfflineDictionary struct {
	db   *sql.DB
	name string
	path string
}

// NewOfflineDict opens an offline dictionary database for the given language pair.
// langPair is like "de-en", "en-zh", "zh-en".
// Returns (nil, nil) if the dictionary file does not exist (no error, just unavailable).
func NewOfflineDict(dbDir, langPair string) (*OfflineDictionary, error) {
	if dbDir == "" {
		return nil, nil
	}
	dbPath := filepath.Join(dbDir, langPair+".db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, nil
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open offline dict %s: %w", dbPath, err)
	}
	if _, err := db.Exec("PRAGMA query_only = 1"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set read-only: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = OFF"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set journal off: %w", err)
	}
	return &OfflineDictionary{
		db:   db,
		name: "offline:" + langPair,
		path: dbPath,
	}, nil
}

// Close closes the database connection.
func (o *OfflineDictionary) Close() error {
	if o.db != nil {
		return o.db.Close()
	}
	return nil
}

// Name returns the unique identifier for this dictionary (e.g. "offline:de-en").
func (o *OfflineDictionary) Name() string {
	return o.name
}

// Path returns the filesystem path to the database file.
func (o *OfflineDictionary) Path() string {
	return o.path
}

// Lookup searches the offline dictionary for the given word.
// Returns (nil, false) if the word is not found or the dictionary is unavailable.
func (o *OfflineDictionary) Lookup(word string) (*TranslationData, bool) {
	if o == nil || o.db == nil {
		return nil, false
	}
	row := o.db.QueryRow("SELECT data FROM entries WHERE query = ?", word)
	var compressed []byte
	if err := row.Scan(&compressed); err != nil {
		return nil, false
	}
	data, err := decompressEntry(compressed)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Stats returns basic statistics about the dictionary.
func (o *OfflineDictionary) Stats() (entries int, size int64, err error) {
	if o == nil || o.db == nil {
		return 0, 0, nil
	}
	row := o.db.QueryRow("SELECT COUNT(*) FROM entries")
	if err := row.Scan(&entries); err != nil {
		return 0, 0, err
	}
	fi, err := os.Stat(o.path)
	if err != nil {
		return entries, 0, nil
	}
	return entries, fi.Size(), nil
}

// DictDir returns the standard directory for offline dictionary databases.
// Default: ~/.config/bl/dict/
func DictDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".config", "bl", "dict"), nil
}

// EnsureDictDir creates the dictionary directory if it does not exist.
func EnsureDictDir() (string, error) {
	dir, err := DictDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create dict dir: %w", err)
	}
	return dir, nil
}

// LangForSource returns the offline dictionary language pair identifier
// for a given DictionarySource name and query text.
func LangForSource(sourceName, text string) string {
	switch sourceName {
	case "woerter-net":
		return "de-en"
	default:
		if IsCJK(text) {
			return "zh-en"
		}
		return "en-zh"
	}
}

// decompressEntry decompresses a zlib-compressed TranslationData JSON blob.
func decompressEntry(compressed []byte) (*TranslationData, error) {
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var data TranslationData
	if err := json.Unmarshal(decompressed, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// CompressEntry compresses a TranslationData into a zlib-compressed JSON blob.
func CompressEntry(data *TranslationData) ([]byte, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(jsonBytes); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// CreateOfflineDict creates a new offline dictionary database from a set of entries.
// entries is a map of word → compressed TranslationData JSON.
// If the file already exists it is overwritten.
func CreateOfflineDict(dbPath string, entries map[string]*TranslationData) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec("PRAGMA journal_mode = OFF"); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS entries (
		query TEXT NOT NULL PRIMARY KEY,
		data  BLOB NOT NULL
	) WITHOUT ROWID`); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO entries (query, data) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for word, td := range entries {
		compressed, err := CompressEntry(td)
		if err != nil {
			continue
		}
		if _, err := stmt.Exec(word, compressed); err != nil {
			tx.Rollback()
			return fmt.Errorf("insert %q: %w", word, err)
		}
	}
	return tx.Commit()
}
