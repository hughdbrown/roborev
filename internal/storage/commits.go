package storage

import (
	"database/sql"
	"time"
)

// GetOrCreateCommit finds or creates a commit record
func (db *DB) GetOrCreateCommit(repoID int64, sha, author, subject string, timestamp time.Time) (*Commit, error) {
	// Try to find existing
	var commit Commit
	var ts, createdAt string
	err := db.QueryRow(`SELECT id, repo_id, sha, author, subject, timestamp, created_at FROM commits WHERE sha = ?`, sha).
		Scan(&commit.ID, &commit.RepoID, &commit.SHA, &commit.Author, &commit.Subject, &ts, &createdAt)
	if err == nil {
		commit.Timestamp, _ = time.Parse(time.RFC3339, ts)
		commit.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		return &commit, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new
	result, err := db.Exec(`INSERT INTO commits (repo_id, sha, author, subject, timestamp) VALUES (?, ?, ?, ?, ?)`,
		repoID, sha, author, subject, timestamp.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &Commit{
		ID:        id,
		RepoID:    repoID,
		SHA:       sha,
		Author:    author,
		Subject:   subject,
		Timestamp: timestamp,
		CreatedAt: time.Now(),
	}, nil
}

// GetCommitBySHA returns a commit by its SHA
func (db *DB) GetCommitBySHA(sha string) (*Commit, error) {
	var commit Commit
	var ts, createdAt string
	err := db.QueryRow(`SELECT id, repo_id, sha, author, subject, timestamp, created_at FROM commits WHERE sha = ?`, sha).
		Scan(&commit.ID, &commit.RepoID, &commit.SHA, &commit.Author, &commit.Subject, &ts, &createdAt)
	if err != nil {
		return nil, err
	}
	commit.Timestamp, _ = time.Parse(time.RFC3339, ts)
	commit.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &commit, nil
}

// GetCommitByID returns a commit by its ID
func (db *DB) GetCommitByID(id int64) (*Commit, error) {
	var commit Commit
	var ts, createdAt string
	err := db.QueryRow(`SELECT id, repo_id, sha, author, subject, timestamp, created_at FROM commits WHERE id = ?`, id).
		Scan(&commit.ID, &commit.RepoID, &commit.SHA, &commit.Author, &commit.Subject, &ts, &createdAt)
	if err != nil {
		return nil, err
	}
	commit.Timestamp, _ = time.Parse(time.RFC3339, ts)
	commit.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &commit, nil
}
