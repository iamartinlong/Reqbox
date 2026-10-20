package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"apimock/internal/store"
)

func (a *api) handleListCollections(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListCollections()
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) handleCreateCollection(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Name == "" {
		body.Name = "New collection"
	}
	item, err := a.store.CreateCollection(body.Name)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *api) handleUpdateCollection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name      *string `json:"name"`
		SortIndex *int    `json:"sortIndex"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := a.store.UpdateCollection(id, body.Name, body.SortIndex)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *api) handleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteCollection(chi.URLParam(r, "id")); err != nil {
		a.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) handleListRequests(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListRequests()
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) handleCreateRequest(w http.ResponseWriter, r *http.Request) {
	var body store.Request
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Name == "" {
		body.Name = "Untitled request"
	}
	if err := a.prepareExposure(&body); err != nil {
		a.fail(w, err)
		return
	}
	item, err := a.store.CreateRequest(body)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// prepareExposure validates a publish request and fills in a slug when one was
// not supplied. Publishing without a slug would otherwise create an endpoint
// nobody can call.
func (a *api) prepareExposure(r *store.Request) error {
	if !r.Exposed {
		// Keep a slug so the request can be republished without retyping it.
		if r.ExposeSlug != "" {
			r.ExposeSlug = trimSlug(r.ExposeSlug)
		}
		return nil
	}

	base := trimSlug(r.ExposeSlug)
	if base == "" {
		base = trimSlug(r.Name)
	}
	if base == "" {
		base = "mock"
	}

	slug := base
	for i := 2; i < 100; i++ {
		taken, err := a.store.SlugTaken(slug, r.ID)
		if err != nil {
			return err
		}
		if !taken {
			break
		}
		// An explicit slug is a deliberate choice, so a collision is an error
		// the user must resolve. A derived one can be made unique silently.
		if trimSlug(r.ExposeSlug) != "" {
			return errSlugTaken
		}
		slug = base + "-" + strconv.Itoa(i)
	}
	r.ExposeSlug = slug
	return nil
}

var errSlugTaken = &slugError{}

type slugError struct{}

func (e *slugError) Error() string { return "mock slug is already in use" }

func (a *api) handleUpdateRequest(w http.ResponseWriter, r *http.Request) {
	var body store.Request
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	body.ID = chi.URLParam(r, "id")

	if err := a.prepareExposure(&body); err != nil {
		a.fail(w, err)
		return
	}
	item, err := a.store.UpdateRequest(body)
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *api) handleDeleteRequest(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DeleteRequest(chi.URLParam(r, "id")); err != nil {
		a.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) handleDuplicateRequest(w http.ResponseWriter, r *http.Request) {
	item, err := a.store.DuplicateRequest(chi.URLParam(r, "id"))
	if err != nil {
		a.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *api) handleReorderRequests(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := a.store.ReorderRequests(body.IDs); err != nil {
		a.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
