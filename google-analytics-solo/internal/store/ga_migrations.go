// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored GA4 report cache. Not generated; safe across regen.
//
// The `report` command caches each successful run here so `trend` can show a
// metric over time offline, without re-querying the API.

package store

import (
	"context"
	"database/sql"
	"fmt"
)

// GAReportRow is one cached report result.
type GAReportRow struct {
	ID       int64  `json:"id"`
	Property string `json:"property"`
	SpecHash string `json:"spec_hash"`
	Dims     string `json:"dims"`
	Metrics  string `json:"metrics"`
	Since    string `json:"since"`
	Until    string `json:"until"`
	RowsJSON string `json:"rows_json"`
	RanAt    string `json:"ran_at"`
}

// EnsureGAReportTable lazily creates the report_run table.
func (s *Store) EnsureGAReportTable(ctx context.Context) error {
	_, err := s.DB().ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS report_run (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    property  TEXT NOT NULL,
    spec_hash TEXT NOT NULL,
    dims      TEXT,
    metrics   TEXT,
    since     TEXT,
    until     TEXT,
    rows_json TEXT,
    ran_at    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_report_run_property ON report_run(property);`)
	if err != nil {
		return fmt.Errorf("creating report_run table: %w", err)
	}
	return nil
}

// InsertGAReport records one report result.
func (s *Store) InsertGAReport(ctx context.Context, r GAReportRow) error {
	if err := s.EnsureGAReportTable(ctx); err != nil {
		return err
	}
	_, err := s.DB().ExecContext(ctx,
		`INSERT INTO report_run(property, spec_hash, dims, metrics, since, until, rows_json, ran_at)
		 VALUES(?,?,?,?,?,?,?,?)`,
		r.Property, r.SpecHash, r.Dims, r.Metrics, r.Since, r.Until, r.RowsJSON, r.RanAt)
	if err != nil {
		return fmt.Errorf("inserting report_run: %w", err)
	}
	return nil
}

// QueryGAReports returns cached runs for a property whose metrics list contains
// the given metric (substring match on the stored csv), newest first.
func (s *Store) QueryGAReports(ctx context.Context, property, metric string) ([]GAReportRow, error) {
	if err := s.EnsureGAReportTable(ctx); err != nil {
		return nil, err
	}
	rows, err := s.DB().QueryContext(ctx,
		`SELECT id, property, spec_hash, dims, metrics, since, until, rows_json, ran_at
		 FROM report_run
		 WHERE property = ? AND (? = '' OR metrics LIKE '%' || ? || '%')
		 ORDER BY ran_at DESC, id DESC`,
		property, metric, metric)
	if err != nil {
		return nil, fmt.Errorf("querying report_run: %w", err)
	}
	defer rows.Close()

	out := make([]GAReportRow, 0)
	for rows.Next() {
		var r GAReportRow
		var dims, metrics, since, until, rowsJSON sql.NullString
		if err := rows.Scan(&r.ID, &r.Property, &r.SpecHash, &dims, &metrics, &since, &until, &rowsJSON, &r.RanAt); err != nil {
			return nil, fmt.Errorf("scanning report_run: %w", err)
		}
		r.Dims, r.Metrics, r.Since, r.Until, r.RowsJSON = dims.String, metrics.String, since.String, until.String, rowsJSON.String
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
