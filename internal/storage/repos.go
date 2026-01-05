package storage

import (
	"database/sql"
	"path/filepath"
	"time"
)

// GetOrCreateRepo finds or creates a repo by its root path
func (db *DB) GetOrCreateRepo(rootPath string) (*Repo, error) {
	// Normalize path
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	// Try to find existing
	var repo Repo
	var createdAt string
	err = db.QueryRow(`SELECT id, root_path, name, created_at FROM repos WHERE root_path = ?`, absPath).
		Scan(&repo.ID, &repo.RootPath, &repo.Name, &createdAt)
	if err == nil {
		repo.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		return &repo, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new
	name := filepath.Base(absPath)
	result, err := db.Exec(`INSERT INTO repos (root_path, name) VALUES (?, ?)`, absPath, name)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &Repo{
		ID:        id,
		RootPath:  absPath,
		Name:      name,
		CreatedAt: time.Now(),
	}, nil
}

// GetRepoByPath returns a repo by its path
func (db *DB) GetRepoByPath(rootPath string) (*Repo, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	var repo Repo
	var createdAt string
	err = db.QueryRow(`SELECT id, root_path, name, created_at FROM repos WHERE root_path = ?`, absPath).
		Scan(&repo.ID, &repo.RootPath, &repo.Name, &createdAt)
	if err != nil {
		return nil, err
	}
	repo.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &repo, nil
}
