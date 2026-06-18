package lyn

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

const projectUpsertSQL = "INSERT INTO projects(path, name, kind, distro, detected_at, updated_at, usage_count, last_launched_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(path) DO UPDATE SET name=excluded.name, kind=excluded.kind, distro=excluded.distro, detected_at=excluded.detected_at, updated_at=excluded.updated_at"

func OpenStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if err := store.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA temp_store=MEMORY",
		"CREATE TABLE IF NOT EXISTS projects (path TEXT PRIMARY KEY, name TEXT NOT NULL, kind TEXT NOT NULL, detected_at TEXT NOT NULL, updated_at TEXT NOT NULL)",
		"ALTER TABLE projects ADD COLUMN usage_count INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE projects ADD COLUMN last_launched_at TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE projects ADD COLUMN distro TEXT NOT NULL DEFAULT ''",
		"CREATE INDEX IF NOT EXISTS idx_projects_kind_name ON projects(kind, name)",
		"CREATE INDEX IF NOT EXISTS idx_projects_usage ON projects(usage_count, last_launched_at)",
		"DELETE FROM projects WHERE kind = 'system-command' OR path LIKE 'lyn:system:%'",
		`DELETE FROM projects WHERE path LIKE '\\wsl%' AND distro = ''`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			if isDuplicateColumnError(err) {
				continue
			}
			return err
		}
	}
	return s.deleteVolatileVSCodeRecentProjects(ctx, runtime.GOOS)
}

func (s *Store) deleteVolatileVSCodeRecentProjects(ctx context.Context, goos string) error {
	_, err := s.db.ExecContext(ctx, volatileVSCodeRecentDeleteSQL(goos))
	return err
}

func volatileVSCodeRecentDeleteSQL(goos string) string {
	if goos == "windows" {
		return "DELETE FROM projects WHERE kind = 'vscode-recent' OR path LIKE 'vscode-remote:%' OR (kind = 'vscode-workspace' AND path LIKE '/%')"
	}
	return "DELETE FROM projects WHERE kind = 'vscode-recent' OR path LIKE 'vscode-remote:%'"
}

func (s *Store) UpsertProjects(ctx context.Context, projects []Project) error {
	return s.writeProjects(ctx, projects, false)
}

func (s *Store) ReplaceProjects(ctx context.Context, projects []Project) error {
	return s.writeProjects(ctx, projects, true)
}

func (s *Store) ReplaceProjectKinds(ctx context.Context, projects []Project, kinds ...string) error {
	if len(kinds) == 0 {
		return s.UpsertProjects(ctx, projects)
	}
	return s.syncProjects(ctx, projects, projectKindSync{
		kinds:        kinds,
		pathSQL:      "INSERT OR IGNORE INTO sync_paths(path) SELECT ? WHERE ? IN (SELECT kind FROM sync_kinds)",
		deleteSQL:    "DELETE FROM projects WHERE kind IN (SELECT kind FROM sync_kinds) AND path NOT IN (SELECT path FROM sync_paths)",
		pathUsesKind: true,
		withStale:    true,
	})
}

func (s *Store) writeProjects(ctx context.Context, projects []Project, removeStale bool) error {
	sync := projectKindSync{}
	if removeStale {
		sync = projectKindSync{
			pathSQL:   "INSERT OR IGNORE INTO sync_paths(path) VALUES(?)",
			deleteSQL: "DELETE FROM projects WHERE path NOT IN (SELECT path FROM sync_paths)",
			withStale: true,
		}
	}
	return s.syncProjects(ctx, projects, sync)
}

type projectKindSync struct {
	kinds        []string
	pathSQL      string
	deleteSQL    string
	pathUsesKind bool
	withStale    bool
}

func (s *Store) syncProjects(ctx context.Context, projects []Project, sync projectKindSync) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if sync.withStale {
		if err := execStatements(ctx, tx,
			"CREATE TEMP TABLE IF NOT EXISTS sync_paths(path TEXT PRIMARY KEY)",
			"DELETE FROM sync_paths",
		); err != nil {
			return err
		}
	}

	if len(sync.kinds) > 0 {
		if err := execStatements(ctx, tx,
			"CREATE TEMP TABLE IF NOT EXISTS sync_kinds(kind TEXT PRIMARY KEY)",
			"DELETE FROM sync_kinds",
		); err != nil {
			return err
		}
		if err := insertSyncKinds(ctx, tx, sync.kinds); err != nil {
			return err
		}
	}

	stmt, err := tx.PrepareContext(ctx, projectUpsertSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	var pathStmt *sql.Stmt
	if sync.withStale {
		pathStmt, err = tx.PrepareContext(ctx, sync.pathSQL)
		if err != nil {
			return err
		}
		defer pathStmt.Close()
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, project := range projects {
		if err := writeProject(ctx, stmt, project, now); err != nil {
			return err
		}
		if pathStmt != nil {
			if err := writeSyncPath(ctx, pathStmt, project, sync.pathUsesKind); err != nil {
				return err
			}
		}
	}

	if sync.deleteSQL != "" {
		if _, err := tx.ExecContext(ctx, sync.deleteSQL); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func execStatements(ctx context.Context, tx *sql.Tx, statements ...string) error {
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func insertSyncKinds(ctx context.Context, tx *sql.Tx, kinds []string) error {
	stmt, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO sync_kinds(kind) VALUES(?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, kind := range kinds {
		if _, err := stmt.ExecContext(ctx, kind); err != nil {
			return err
		}
	}
	return nil
}

func writeProject(ctx context.Context, stmt *sql.Stmt, project Project, updatedAt string) error {
	_, err := stmt.ExecContext(ctx, project.Path, project.Name, project.Kind, project.Distro, formatTime(project.DetectedAt), updatedAt, project.UsageCount, formatTime(project.LastLaunchedAt))
	return err
}

func writeSyncPath(ctx context.Context, stmt *sql.Stmt, project Project, includeKind bool) error {
	if includeKind {
		_, err := stmt.ExecContext(ctx, project.Path, project.Kind)
		return err
	}
	_, err := stmt.ExecContext(ctx, project.Path)
	return err
}

func (s *Store) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT name, path, kind, distro, detected_at, usage_count, last_launched_at FROM projects ORDER BY usage_count DESC, last_launched_at DESC, updated_at DESC, name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Project
	for rows.Next() {
		var item Project
		var detectedAt string
		var lastLaunchedAt string
		if err := rows.Scan(&item.Name, &item.Path, &item.Kind, &item.Distro, &detectedAt, &item.UsageCount, &lastLaunchedAt); err != nil {
			return nil, err
		}
		item.DetectedAt, _ = time.Parse(time.RFC3339Nano, detectedAt)
		item.LastLaunchedAt, _ = time.Parse(time.RFC3339Nano, lastLaunchedAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) RecordLaunch(ctx context.Context, path string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, "UPDATE projects SET usage_count = usage_count + 1, last_launched_at = ?, updated_at = ? WHERE path = ?", now, now, path)
	return err
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func isDuplicateColumnError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}
