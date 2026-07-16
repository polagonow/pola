package migrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"ariga.io/atlas/sql/migrate"
)

// RevisionReadWriter is a dialect-aware migrate.RevisionReadWriter that stores
// migration history in an atlas_schema_revisions table. It supports sqlite,
// postgresql, and mysql so the same revision bookkeeping is used regardless of
// which database the app or CLI targets.
type RevisionReadWriter struct {
	db      *sql.DB
	adapter string
}

// Ensure *RevisionReadWriter satisfies the Atlas interface.
var _ migrate.RevisionReadWriter = (*RevisionReadWriter)(nil)

// NewRevisionReadWriter returns a RevisionReadWriter for the given adapter
// ("sqlite", "postgresql", or "mysql").
func NewRevisionReadWriter(db *sql.DB, adapter string) *RevisionReadWriter {
	return &RevisionReadWriter{db: db, adapter: adapter}
}

// placeholder renders the nth (1-based) bind placeholder for the adapter.
// Postgres uses $1, $2, …; sqlite and mysql use ?.
func (rw *RevisionReadWriter) placeholder(n int) string {
	if rw.adapter == "postgresql" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

// placeholders renders a comma-separated list of count placeholders.
func (rw *RevisionReadWriter) placeholders(count int) string {
	ps := make([]string, count)
	for i := range ps {
		ps[i] = rw.placeholder(i + 1)
	}
	return strings.Join(ps, ", ")
}

// Init creates the atlas_schema_revisions table if it does not already exist.
func (rw *RevisionReadWriter) Init(ctx context.Context) error {
	var ddl string
	switch rw.adapter {
	case "postgresql":
		ddl = `CREATE TABLE IF NOT EXISTS atlas_schema_revisions (
			version TEXT PRIMARY KEY,
			description TEXT NOT NULL DEFAULT '',
			type BIGINT NOT NULL DEFAULT 0,
			applied BIGINT NOT NULL DEFAULT 0,
			total BIGINT NOT NULL DEFAULT 0,
			executed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			execution_time BIGINT NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '',
			error_stmt TEXT NOT NULL DEFAULT '',
			hash TEXT NOT NULL DEFAULT '',
			operator_version TEXT NOT NULL DEFAULT ''
		)`
	case "mysql":
		ddl = "CREATE TABLE IF NOT EXISTS atlas_schema_revisions (" +
			"version VARCHAR(255) PRIMARY KEY," +
			"description VARCHAR(255) NOT NULL DEFAULT ''," +
			"type BIGINT NOT NULL DEFAULT 0," +
			"applied BIGINT NOT NULL DEFAULT 0," +
			"total BIGINT NOT NULL DEFAULT 0," +
			"executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP," +
			"execution_time BIGINT NOT NULL DEFAULT 0," +
			"error TEXT," +
			"error_stmt TEXT," +
			"hash VARCHAR(255) NOT NULL DEFAULT ''," +
			"operator_version VARCHAR(255) NOT NULL DEFAULT ''" +
			")"
	default: // sqlite
		ddl = `CREATE TABLE IF NOT EXISTS atlas_schema_revisions (
			version TEXT PRIMARY KEY,
			description TEXT NOT NULL DEFAULT '',
			type INTEGER NOT NULL DEFAULT 0,
			applied INTEGER NOT NULL DEFAULT 0,
			total INTEGER NOT NULL DEFAULT 0,
			executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			execution_time INTEGER NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '',
			error_stmt TEXT NOT NULL DEFAULT '',
			hash TEXT NOT NULL DEFAULT '',
			operator_version TEXT NOT NULL DEFAULT ''
		)`
	}
	_, err := rw.db.ExecContext(ctx, ddl)
	return err
}

// Ident implements migrate.RevisionReadWriter.
func (rw *RevisionReadWriter) Ident() *migrate.TableIdent {
	return &migrate.TableIdent{Name: "atlas_schema_revisions"}
}

const revisionColumns = "version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, operator_version"

// ReadRevisions implements migrate.RevisionReadWriter.
func (rw *RevisionReadWriter) ReadRevisions(ctx context.Context) ([]*migrate.Revision, error) {
	rows, err := rw.db.QueryContext(ctx, "SELECT "+revisionColumns+" FROM atlas_schema_revisions ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revs []*migrate.Revision
	for rows.Next() {
		r, err := scanRevision(rows.Scan)
		if err != nil {
			return nil, err
		}
		revs = append(revs, r)
	}
	return revs, rows.Err()
}

// ReadRevision implements migrate.RevisionReadWriter.
func (rw *RevisionReadWriter) ReadRevision(ctx context.Context, version string) (*migrate.Revision, error) {
	query := "SELECT " + revisionColumns + " FROM atlas_schema_revisions WHERE version = " + rw.placeholder(1)
	row := rw.db.QueryRowContext(ctx, query, version)
	r, err := scanRevision(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, migrate.ErrRevisionNotExist
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// scanRevision decodes a single revision row from the given scan function.
func scanRevision(scan func(dest ...any) error) (*migrate.Revision, error) {
	var r migrate.Revision
	var execTime int64
	if err := scan(&r.Version, &r.Description, &r.Type, &r.Applied, &r.Total, &r.ExecutedAt, &execTime, &r.Error, &r.ErrorStmt, &r.Hash, &r.OperatorVersion); err != nil {
		return nil, err
	}
	r.ExecutionTime = time.Duration(execTime)
	return &r, nil
}

// WriteRevision implements migrate.RevisionReadWriter, upserting the revision.
func (rw *RevisionReadWriter) WriteRevision(ctx context.Context, r *migrate.Revision) error {
	values := "(" + rw.placeholders(11) + ")"
	var query string
	switch rw.adapter {
	case "postgresql":
		query = "INSERT INTO atlas_schema_revisions (" + revisionColumns + ") VALUES " + values +
			` ON CONFLICT (version) DO UPDATE SET
				description = EXCLUDED.description,
				type = EXCLUDED.type,
				applied = EXCLUDED.applied,
				total = EXCLUDED.total,
				executed_at = EXCLUDED.executed_at,
				execution_time = EXCLUDED.execution_time,
				error = EXCLUDED.error,
				error_stmt = EXCLUDED.error_stmt,
				hash = EXCLUDED.hash,
				operator_version = EXCLUDED.operator_version`
	case "mysql":
		query = "INSERT INTO atlas_schema_revisions (" + revisionColumns + ") VALUES " + values +
			" ON DUPLICATE KEY UPDATE " +
			"description = VALUES(description)," +
			"type = VALUES(type)," +
			"applied = VALUES(applied)," +
			"total = VALUES(total)," +
			"executed_at = VALUES(executed_at)," +
			"execution_time = VALUES(execution_time)," +
			"error = VALUES(error)," +
			"error_stmt = VALUES(error_stmt)," +
			"hash = VALUES(hash)," +
			"operator_version = VALUES(operator_version)"
	default: // sqlite
		query = "INSERT OR REPLACE INTO atlas_schema_revisions (" + revisionColumns + ") VALUES " + values
	}
	_, err := rw.db.ExecContext(ctx, query,
		r.Version, r.Description, r.Type, r.Applied, r.Total, r.ExecutedAt, int64(r.ExecutionTime), r.Error, r.ErrorStmt, r.Hash, r.OperatorVersion)
	return err
}

// DeleteRevision implements migrate.RevisionReadWriter.
func (rw *RevisionReadWriter) DeleteRevision(ctx context.Context, version string) error {
	_, err := rw.db.ExecContext(ctx, "DELETE FROM atlas_schema_revisions WHERE version = "+rw.placeholder(1), version)
	return err
}
