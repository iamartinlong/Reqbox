package store

import (
	"database/sql"
	"errors"
)

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("not found")

// ListCollections returns all collections ordered by sort_index, name.
func (s *Store) ListCollections() ([]Collection, error) {
	rows, err := s.db.Query(`SELECT id, name, sort_index, created_at, updated_at FROM collections ORDER BY sort_index, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Collection, 0)
	for rows.Next() {
		var c Collection
		var created, updated string
		if err := rows.Scan(&c.ID, &c.Name, &c.SortIndex, &created, &updated); err != nil {
			return nil, err
		}
		c.CreatedAt, c.UpdatedAt = parseTime(created), parseTime(updated)
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateCollection inserts a collection.
func (s *Store) CreateCollection(name string) (Collection, error) {
	now := now()
	c := Collection{ID: newID(), Name: name, CreatedAt: now, UpdatedAt: now}

	var maxOrder sql.NullInt64
	_ = s.db.QueryRow(`SELECT MAX(sort_index) FROM collections`).Scan(&maxOrder)
	c.SortIndex = int(maxOrder.Int64) + 1

	_, err := s.db.Exec(`INSERT INTO collections (id, name, sort_index, created_at, updated_at) VALUES (?,?,?,?,?)`,
		c.ID, c.Name, c.SortIndex, formatTime(c.CreatedAt), formatTime(c.UpdatedAt))
	if err != nil {
		return Collection{}, err
	}
	return c, nil
}

// UpdateCollection renames or reorders a collection.
func (s *Store) UpdateCollection(id string, name *string, sortIndex *int) (Collection, error) {
	c, err := s.GetCollection(id)
	if err != nil {
		return Collection{}, err
	}
	if name != nil {
		c.Name = *name
	}
	if sortIndex != nil {
		c.SortIndex = *sortIndex
	}
	c.UpdatedAt = now()
	_, err = s.db.Exec(`UPDATE collections SET name = ?, sort_index = ?, updated_at = ? WHERE id = ?`,
		c.Name, c.SortIndex, formatTime(c.UpdatedAt), c.ID)
	if err != nil {
		return Collection{}, err
	}
	return c, nil
}

// GetCollection loads one collection.
func (s *Store) GetCollection(id string) (Collection, error) {
	var c Collection
	var created, updated string
	err := s.db.QueryRow(`SELECT id, name, sort_index, created_at, updated_at FROM collections WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.SortIndex, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Collection{}, ErrNotFound
	}
	if err != nil {
		return Collection{}, err
	}
	c.CreatedAt, c.UpdatedAt = parseTime(created), parseTime(updated)
	return c, nil
}

// DeleteCollection removes a collection; requests inside keep existing but lose the link.
func (s *Store) DeleteCollection(id string) error {
	res, err := s.db.Exec(`DELETE FROM collections WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
