package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/mattn/go-sqlite3"
)

var ErrNotFound = errors.New("not found requested data")

const createTables = `
CREATE TABLE IF NOT EXISTS sessions (
	github_user_id INTEGER NOT NULL,
	session_id BLOB NOT NULL,
	created_at INTEGER NOT NULL
) STRICT;

CREATE TABLE IF NOT EXISTS shares (
	github_user_id INTEGER NOT NULL,
	path TEXT NOT NULL,
	chart BLOB NOT NULL,
	created_at INTEGER NOT NULL
) STRICT;

CREATE UNIQUE INDEX IF NOT EXISTS shares_unique_path ON shares (path);
`

type SqliteStorage struct {
	sql *sql.DB
}

func NewSqliteStorage(path string) (SqliteStorage, error) {
	sql, err := sql.Open("sqlite3", path)
	if err != nil {
		return SqliteStorage{}, fmt.Errorf("failed while oppening sqlite database: %v: %v", path, err)
	}

	if _, err := sql.Exec(createTables); err != nil {
		return SqliteStorage{}, fmt.Errorf("failed while creating default schema: %v", err)
	}

	return SqliteStorage{
		sql: sql,
	}, nil
}

type Session struct {
	GithubUserID uint64
	SessionID    [32]byte
}

func (d *SqliteStorage) StoreSession(s *Session) error {
	_, err := d.sql.Exec("INSERT INTO sessions VALUES(?, ?, UNIXEPOCH())", s.GithubUserID, s.SessionID[:])
	return err
}

func (d *SqliteStorage) IsSessionValid(s *Session) error {
	res, err := d.sql.Query("SELECT * FROM sessions WHERE github_user_id = ? AND session_id = ?", s.GithubUserID, s.SessionID[:])
	if err != nil {
		return err
	}
	defer res.Close()
	if !res.Next() {
		return ErrNotFound
	}
	return nil
}

func (d *SqliteStorage) RemoveSession(s *Session) error {
	_, err := d.sql.Exec("DELETE FROM sessions WHERE github_user_id = ? AND session_id = ?", s.GithubUserID, s.SessionID[:])
	return err
}

func (d *SqliteStorage) IsPathAvail(path string) (bool, error) {
	res, err := d.sql.Query("SELECT * FROM shares WHERE path = ?", path)
	if err != nil {
		return false, err
	}
	defer res.Close()
	return !res.Next(), nil
}

type Share struct {
	GithubUserID uint64
	Path         string
	Chart        []byte
}

var createShareMutex sync.Mutex
var ErrTooMuchShares = errors.New("too much shares")

func (d *SqliteStorage) CreateShare(share *Share, maxSharesPerUser int) (bool, error) {
	// Not sure how sqlite implements sql transactins, so using a mutex to ensure that
	// there is no race condition (max shares count check not running concurrently).
	createShareMutex.Lock()
	defer createShareMutex.Unlock()

	var count int
	row := d.sql.QueryRow("SELECT COUNT(*) FROM shares WHERE github_user_id = ?", share.GithubUserID)
	if err := row.Scan(&count); err != nil {
		return false, err
	}

	if count >= maxSharesPerUser {
		return false, ErrTooMuchShares
	}

	_, err := d.sql.Exec("INSERT INTO shares VALUES(?, ?, ?, UNIXEPOCH())", share.GithubUserID, share.Path, share.Chart)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return false, nil
		}
		return false, err
	}
	return true, err
}

func (d *SqliteStorage) GetShare(path string) (*Share, error) {
	row := d.sql.QueryRow("SELECT github_user_id, chart FROM shares WHERE path = ?", path)

	ret := &Share{Path: path}
	if err := row.Scan(&ret.GithubUserID, &ret.Chart); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return ret, nil
}

func (d *SqliteStorage) GetUserShares(githubUserID uint64) ([]Share, error) {
	res, err := d.sql.Query("SELECT path, chart FROM shares WHERE github_user_id = ?", githubUserID)
	if err != nil {
		return nil, err
	}

	shares := make([]Share, 0, 8)
	for res.Next() {
		share := Share{GithubUserID: githubUserID}
		// TODO: is this required for correct error handling, doesn't the Err() method below hadle that too.??
		if err := res.Scan(&share.Path, &share.Chart); err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}

	if err := res.Err(); err != nil {
		return nil, err
	}

	return shares, nil
}

func (d *SqliteStorage) RemoveShare(path string, githubUserID uint64) error {
	_, err := d.sql.Exec("DELETE FROM shares WHERE github_user_id = ? AND path = ?", githubUserID, path)
	if err != nil {
		return err
	}
	return nil
}
