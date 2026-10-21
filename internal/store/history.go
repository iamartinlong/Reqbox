package store

import (
	"database/sql"
	"encoding/json"
	"errors"
)

// InsertHistory records an execution. Request snapshots keep history readable
// after the originating request is edited or deleted.
func (s *Store) InsertHistory(h HistoryEntry) (HistoryEntry, error) {
	h.ID = newID()
	h.CreatedAt = now()
	_, err := s.db.Exec(`INSERT INTO history (id, request_id, name, method, url, status, duration_ms,
		size_bytes, ok, error, request_snapshot, response_headers, response_body, body_encoding, timing, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		h.ID, h.RequestID, h.Name, h.Method, h.URL, h.Status, h.DurationMs, h.SizeBytes,
		boolInt(h.OK), h.Error, h.RequestSnapshot, h.ResponseHeaders, h.ResponseBody,
		h.BodyEncoding, h.Timing, formatTime(h.CreatedAt))
	if err != nil {
		return HistoryEntry{}, err
	}
	return h, nil
}

// ListHistory returns newest-first history entries, optionally filtered.
func (s *Store) ListHistory(search string, limit int) ([]HistoryEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT id, request_id, name, method, url, status, duration_ms, size_bytes, ok, error,
		request_snapshot, response_headers, response_body, body_encoding, timing, created_at
		FROM history`
	args := []any{}
	if search != "" {
		query += ` WHERE name LIKE ? OR url LIKE ? OR method LIKE ?`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]HistoryEntry, 0, limit)
	for rows.Next() {
		h, err := scanHistory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func scanHistory(sc interface{ Scan(...any) error }) (HistoryEntry, error) {
	var h HistoryEntry
	var ok int
	var created string
	err := sc.Scan(&h.ID, &h.RequestID, &h.Name, &h.Method, &h.URL, &h.Status, &h.DurationMs,
		&h.SizeBytes, &ok, &h.Error, &h.RequestSnapshot, &h.ResponseHeaders, &h.ResponseBody,
		&h.BodyEncoding, &h.Timing, &created)
	if err != nil {
		return HistoryEntry{}, err
	}
	h.OK = ok == 1
	h.CreatedAt = parseTime(created)
	return h, nil
}

// GetHistory loads one entry including captured bodies.
func (s *Store) GetHistory(id string) (HistoryEntry, error) {
	row := s.db.QueryRow(`SELECT id, request_id, name, method, url, status, duration_ms, size_bytes, ok,
		error, request_snapshot, response_headers, response_body, body_encoding, timing, created_at
		FROM history WHERE id = ?`, id)
	h, err := scanHistory(row)
	if errors.Is(err, sql.ErrNoRows) {
		return HistoryEntry{}, ErrNotFound
	}
	return h, err
}

// DeleteHistory removes a single entry.
func (s *Store) DeleteHistory(id string) error {
	res, err := s.db.Exec(`DELETE FROM history WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ClearHistory deletes all entries and returns how many rows were removed.
func (s *Store) ClearHistory() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM history`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// PruneHistory keeps at most keep newest entries.
func (s *Store) PruneHistory(keep int) error {
	if keep < 0 {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM history WHERE id NOT IN (
		SELECT id FROM history ORDER BY created_at DESC LIMIT ?)`, keep)
	return err
}

// -- environments ---------------------------------------------------------

func (s *Store) ListEnvironments() ([]Environment, error) {
	rows, err := s.db.Query(`SELECT id, name, sort_index, created_at, updated_at FROM environments ORDER BY sort_index, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Environment, 0)
	for rows.Next() {
		var e Environment
		var created, updated string
		if err := rows.Scan(&e.ID, &e.Name, &e.SortIndex, &created, &updated); err != nil {
			return nil, err
		}
		e.CreatedAt, e.UpdatedAt = parseTime(created), parseTime(updated)
		vars, err := s.listEnvVars(e.ID)
		if err != nil {
			return nil, err
		}
		e.Variables = vars
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) listEnvVars(envID string) ([]EnvVar, error) {
	rows, err := s.db.Query(`SELECT id, key, value, enabled, secret FROM env_vars WHERE environment_id = ? ORDER BY sort_index`, envID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]EnvVar, 0)
	for rows.Next() {
		var v EnvVar
		var enabled, secret int
		if err := rows.Scan(&v.ID, &v.Key, &v.Value, &enabled, &secret); err != nil {
			return nil, err
		}
		v.Enabled, v.Secret = enabled == 1, secret == 1
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) GetEnvironment(id string) (Environment, error) {
	var e Environment
	var created, updated string
	err := s.db.QueryRow(`SELECT id, name, sort_index, created_at, updated_at FROM environments WHERE id = ?`, id).
		Scan(&e.ID, &e.Name, &e.SortIndex, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Environment{}, ErrNotFound
	}
	if err != nil {
		return Environment{}, err
	}
	e.CreatedAt, e.UpdatedAt = parseTime(created), parseTime(updated)
	e.Variables, err = s.listEnvVars(e.ID)
	return e, err
}

func (s *Store) CreateEnvironment(name string, vars []EnvVar) (Environment, error) {
	now := now()
	e := Environment{ID: newID(), Name: name, CreatedAt: now, UpdatedAt: now}
	var maxOrder sql.NullInt64
	_ = s.db.QueryRow(`SELECT MAX(sort_index) FROM environments`).Scan(&maxOrder)
	e.SortIndex = int(maxOrder.Int64) + 1

	tx, err := s.db.Begin()
	if err != nil {
		return Environment{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO environments (id, name, sort_index, created_at, updated_at) VALUES (?,?,?,?,?)`,
		e.ID, e.Name, e.SortIndex, formatTime(now), formatTime(now)); err != nil {
		return Environment{}, err
	}
	for i, v := range vars {
		if _, err := tx.Exec(`INSERT INTO env_vars (id, environment_id, key, value, enabled, secret, sort_index)
			VALUES (?,?,?,?,?,?,?)`, newID(), e.ID, v.Key, v.Value, boolInt(v.Enabled), boolInt(v.Secret), i); err != nil {
			return Environment{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Environment{}, err
	}
	return s.GetEnvironment(e.ID)
}

func (s *Store) UpdateEnvironment(id, name string, vars []EnvVar) (Environment, error) {
	if _, err := s.GetEnvironment(id); err != nil {
		return Environment{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Environment{}, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE environments SET name = ?, updated_at = ? WHERE id = ?`, name, formatTime(now()), id); err != nil {
		return Environment{}, err
	}
	if _, err := tx.Exec(`DELETE FROM env_vars WHERE environment_id = ?`, id); err != nil {
		return Environment{}, err
	}
	for i, v := range vars {
		if _, err := tx.Exec(`INSERT INTO env_vars (id, environment_id, key, value, enabled, secret, sort_index)
			VALUES (?,?,?,?,?,?,?)`, newID(), id, v.Key, v.Value, boolInt(v.Enabled), boolInt(v.Secret), i); err != nil {
			return Environment{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Environment{}, err
	}
	return s.GetEnvironment(id)
}

func (s *Store) DeleteEnvironment(id string) error {
	res, err := s.db.Exec(`DELETE FROM environments WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// -- settings -------------------------------------------------------------

func (s *Store) GetSetting(key, fallback string) string {
	var v string
	if err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v); err != nil {
		return fallback
	}
	return v
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO settings (key, value) VALUES (?,?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// -- export / import ------------------------------------------------------

// Snap returns everything needed to rebuild the workspace.
func (s *Store) Snap() (Export, error) {
	collections, err := s.ListCollections()
	if err != nil {
		return Export{}, err
	}
	requests, err := s.ListRequests()
	if err != nil {
		return Export{}, err
	}
	envs, err := s.ListEnvironments()
	if err != nil {
		return Export{}, err
	}
	return Export{
		Version:             "1",
		ExportedAt:          now(),
		Collections:         collections,
		Requests:            requests,
		Environments:        envs,
		ActiveEnvironmentID: s.GetSetting("activeEnvironmentId", ""),
	}, nil
}

// Restore replaces the workspace with the supplied snapshot.
func (s *Store) Restore(data Export) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range []string{
		`DELETE FROM env_vars`, `DELETE FROM environments`, `DELETE FROM requests`, `DELETE FROM collections`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	for _, c := range data.Collections {
		created := c.CreatedAt
		if created.IsZero() {
			created = now()
		}
		updated := c.UpdatedAt
		if updated.IsZero() {
			updated = created
		}
		if _, err := tx.Exec(`INSERT INTO collections (id, name, sort_index, created_at, updated_at) VALUES (?,?,?,?,?)`,
			c.ID, c.Name, c.SortIndex, formatTime(created), formatTime(updated)); err != nil {
			return err
		}
	}

	for _, r := range data.Requests {
		s.normalizeRequest(&r)
		created := r.CreatedAt
		if created.IsZero() {
			created = now()
		}
		updated := r.UpdatedAt
		if updated.IsZero() {
			updated = created
		}
		params, _ := json.Marshal(r.Params)
		headers, _ := json.Marshal(r.Headers)
		form, _ := json.Marshal(r.Form)
		auth, _ := json.Marshal(r.Auth)
		if _, err := tx.Exec(`INSERT INTO requests (`+requestColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			r.ID, r.CollectionID, r.Name, r.Method, r.URL, string(params), string(headers), r.BodyType,
			r.Body, string(form), r.AuthType, string(auth), r.SortIndex, r.ExposeSlug, r.Exposed,
			formatTime(created), formatTime(updated)); err != nil {
			return err
		}
	}

	for _, e := range data.Environments {
		created := e.CreatedAt
		if created.IsZero() {
			created = now()
		}
		updated := e.UpdatedAt
		if updated.IsZero() {
			updated = created
		}
		if _, err := tx.Exec(`INSERT INTO environments (id, name, sort_index, created_at, updated_at) VALUES (?,?,?,?,?)`,
			e.ID, e.Name, e.SortIndex, formatTime(created), formatTime(updated)); err != nil {
			return err
		}
		for i, v := range e.Variables {
			if _, err := tx.Exec(`INSERT INTO env_vars (id, environment_id, key, value, enabled, secret, sort_index)
				VALUES (?,?,?,?,?,?,?)`, newID(), e.ID, v.Key, v.Value, boolInt(v.Enabled), boolInt(v.Secret), i); err != nil {
				return err
			}
		}
	}

	if data.ActiveEnvironmentID != "" {
		if _, err := tx.Exec(`INSERT INTO settings (key, value) VALUES ('activeEnvironmentId', ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`, data.ActiveEnvironmentID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Stats returns small counters for the sidebar footer.
func (s *Store) Stats() (map[string]int, error) {
	out := map[string]int{}
	for key, query := range map[string]string{
		"collections":  `SELECT COUNT(*) FROM collections`,
		"requests":     `SELECT COUNT(*) FROM requests`,
		"history":      `SELECT COUNT(*) FROM history`,
		"environments": `SELECT COUNT(*) FROM environments`,
	} {
		var n int
		if err := s.db.QueryRow(query).Scan(&n); err != nil {
			return nil, err
		}
		out[key] = n
	}
	return out, nil
}
