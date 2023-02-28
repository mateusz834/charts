package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"
)

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

func (d *SqliteStorage) IsSessionValid(s *Session) (bool, error) {
	//TODO: handle time validity of session.
	res, err := d.sql.Query("SELECT * FROM sessions WHERE github_user_id = ? AND session_id = ?", s.GithubUserID, s.SessionID[:])
	if err != nil {
		return false, err
	}
	defer res.Close()
	return res.Next(), nil
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

func (d *SqliteStorage) CreateShare(share *Share) (bool, error) {
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
	// TODO: handle non-exisitng paths somehow.
	// TOOD: handle res.Err()
	res, err := d.sql.Query("SELECT github_user_id, chart FROM shares WHERE path = ?", path)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	if !res.Next() {
		return nil, fmt.Errorf("no entry in database for requested path")
	}

	ret := &Share{Path: path}
	if err := res.Scan(&ret.GithubUserID, &ret.Chart); err != nil {
		return nil, err
	}
	return ret, nil
}
