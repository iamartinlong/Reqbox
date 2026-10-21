package store

import (
	"database/sql"
	"encoding/json"
	"errors"
)

const requestColumns = `id, collection_id, name, method, url, params, headers, body_type, body, form,
	auth_type, auth, sort_index, expose_slug, exposed, created_at, updated_at`

func scanRequest(sc interface{ Scan(...any) error }) (Request, error) {
	var r Request
	var collectionID sql.NullString
	var params, headers, form, auth string
	var created, updated string
	err := sc.Scan(&r.ID, &collectionID, &r.Name, &r.Method, &r.URL, &params, &headers,
		&r.BodyType, &r.Body, &form, &r.AuthType, &auth, &r.SortIndex, &r.ExposeSlug, &r.Exposed,
		&created, &updated)
	if err != nil {
		return Request{}, err
	}
	if collectionID.Valid && collectionID.String != "" {
		id := collectionID.String
		r.CollectionID = &id
	}
	r.Params = decodeKV(params)
	r.Headers = decodeKV(headers)
	r.Form = decodeKV(form)
	_ = json.Unmarshal([]byte(auth), &r.Auth)
	if r.Params == nil {
		r.Params = []KV{}
	}
	if r.Headers == nil {
		r.Headers = []KV{}
	}
	if r.Form == nil {
		r.Form = []KV{}
	}
	r.CreatedAt, r.UpdatedAt = parseTime(created), parseTime(updated)
	return r, nil
}

func decodeKV(raw string) []KV {
	out := []KV{}
	if raw == "" {
		return out
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []KV{}
	}
	return out
}

func (s *Store) normalizeRequest(r *Request) {
	if r.Method == "" {
		r.Method = "GET"
	}
	if r.BodyType == "" {
		r.BodyType = "none"
	}
	if r.AuthType == "" {
		r.AuthType = "none"
	}
	if r.Params == nil {
		r.Params = []KV{}
	}
	if r.Headers == nil {
		r.Headers = []KV{}
	}
	if r.Form == nil {
		r.Form = []KV{}
	}
}

// ListRequests returns every saved request.
func (s *Store) ListRequests() ([]Request, error) {
	rows, err := s.db.Query(`SELECT ` + requestColumns + ` FROM requests ORDER BY sort_index, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Request, 0)
	for rows.Next() {
		r, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetRequest loads one request.
func (s *Store) GetRequest(id string) (Request, error) {
	row := s.db.QueryRow(`SELECT `+requestColumns+` FROM requests WHERE id = ?`, id)
	r, err := scanRequest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	return r, err
}

// GetRequestBySlug loads an exposed request by its public slug.
func (s *Store) GetRequestBySlug(slug string) (Request, error) {
	row := s.db.QueryRow(`SELECT `+requestColumns+` FROM requests WHERE expose_slug = ? AND exposed = 1 AND expose_slug <> ''`, slug)
	r, err := scanRequest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	return r, err
}

// SlugTaken reports whether a slug is already used by another request.
func (s *Store) SlugTaken(slug, exceptID string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE expose_slug = ? AND id <> ?`, slug, exceptID).Scan(&n)
	return n > 0, err
}

// CreateRequest inserts a request and assigns a sort order.
func (s *Store) CreateRequest(r Request) (Request, error) {
	s.normalizeRequest(&r)
	r.ID = newID()
	now := now()
	r.CreatedAt, r.UpdatedAt = now, now

	var maxOrder sql.NullInt64
	_ = s.db.QueryRow(`SELECT MAX(sort_index) FROM requests`).Scan(&maxOrder)
	r.SortIndex = int(maxOrder.Int64) + 1

	if err := s.writeRequest(r, true); err != nil {
		return Request{}, err
	}
	return r, nil
}

// UpdateRequest replaces all mutable fields of a request. setCollection says
// whether CollectionID should be applied; when false the existing link is kept.
func (s *Store) UpdateRequest(r Request, setCollection bool) (Request, error) {
	existing, err := s.GetRequest(r.ID)
	if err != nil {
		return Request{}, err
	}
	s.normalizeRequest(&r)
	if !setCollection {
		r.CollectionID = existing.CollectionID
	}
	r.CreatedAt = existing.CreatedAt
	r.UpdatedAt = now()
	if r.SortIndex == 0 {
		r.SortIndex = existing.SortIndex
	}
	if err := s.writeRequest(r, false); err != nil {
		return Request{}, err
	}
	return r, nil
}

func (s *Store) writeRequest(r Request, insert bool) error {
	params, _ := json.Marshal(r.Params)
	headers, _ := json.Marshal(r.Headers)
	form, _ := json.Marshal(r.Form)
	auth, _ := json.Marshal(r.Auth)

	if insert {
		_, err := s.db.Exec(`INSERT INTO requests (`+requestColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			r.ID, r.CollectionID, r.Name, r.Method, r.URL, string(params), string(headers), r.BodyType,
			r.Body, string(form), r.AuthType, string(auth), r.SortIndex, r.ExposeSlug, r.Exposed,
			formatTime(r.CreatedAt), formatTime(r.UpdatedAt))
		return err
	}
	_, err := s.db.Exec(`UPDATE requests SET collection_id=?, name=?, method=?, url=?, params=?, headers=?,
		body_type=?, body=?, form=?, auth_type=?, auth=?, sort_index=?, expose_slug=?, exposed=?, updated_at=?
		WHERE id = ?`,
		r.CollectionID, r.Name, r.Method, r.URL, string(params), string(headers), r.BodyType, r.Body,
		string(form), r.AuthType, string(auth), r.SortIndex, r.ExposeSlug, r.Exposed,
		formatTime(r.UpdatedAt), r.ID)
	return err
}

// DeleteRequest removes a request. History rows are kept for the audit trail.
func (s *Store) DeleteRequest(id string) error {
	res, err := s.db.Exec(`DELETE FROM requests WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DuplicateRequest copies a request with a new id and "copy" suffix.
func (s *Store) DuplicateRequest(id string) (Request, error) {
	r, err := s.GetRequest(id)
	if err != nil {
		return Request{}, err
	}
	r.ID = ""
	r.Name = r.Name + " copy"
	r.ExposeSlug = ""
	r.Exposed = false
	return s.CreateRequest(r)
}

// ReorderRequests applies a list of ids in display order.
func (s *Store) ReorderRequests(ids []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, id := range ids {
		if _, err := tx.Exec(`UPDATE requests SET sort_index = ? WHERE id = ?`, i+1, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
