package store

import (
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestCollectionLifecycle(t *testing.T) {
	s := testStore(t)

	col, err := s.CreateCollection("Payments")
	if err != nil {
		t.Fatal(err)
	}
	if col.ID == "" || col.Name != "Payments" {
		t.Fatalf("unexpected collection: %+v", col)
	}

	all, err := s.ListCollections()
	if err != nil || len(all) != 1 {
		t.Fatalf("list = %v, %v", all, err)
	}

	name := "Billing"
	updated, err := s.UpdateCollection(col.ID, &name, nil)
	if err != nil || updated.Name != "Billing" {
		t.Fatalf("update = %+v, %v", updated, err)
	}

	if err := s.DeleteCollection(col.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetCollection(col.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRequestRoundTrip(t *testing.T) {
	s := testStore(t)
	col, _ := s.CreateCollection("Group")

	created, err := s.CreateRequest(Request{
		CollectionID: &col.ID,
		Name:         "Create user",
		Method:       "POST",
		URL:          "https://api.test/users",
		Params:       []KV{{Key: "q", Value: "1", Enabled: true}},
		Headers:      []KV{{Key: "Accept", Value: "application/json", Enabled: true}},
		BodyType:     "json",
		Body:         `{"name":"Ada"}`,
		AuthType:     "bearer",
		Auth:         Auth{Token: "t"},
		ExposeSlug:   "create-user",
		Exposed:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetRequest(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Params) != 1 || got.Params[0].Key != "q" {
		t.Errorf("params did not round-trip: %+v", got.Params)
	}
	if got.Auth.Token != "t" {
		t.Errorf("auth did not round-trip: %+v", got.Auth)
	}
	if !got.Exposed || got.ExposeSlug != "create-user" {
		t.Errorf("exposure did not round-trip: %+v", got)
	}

	bySlug, err := s.GetRequestBySlug("create-user")
	if err != nil || bySlug.ID != created.ID {
		t.Fatalf("slug lookup = %+v, %v", bySlug, err)
	}

	taken, err := s.SlugTaken("create-user", "")
	if err != nil || !taken {
		t.Errorf("slug should be taken: %v %v", taken, err)
	}
	taken, _ = s.SlugTaken("create-user", created.ID)
	if taken {
		t.Error("slug should not be taken by its own request")
	}
}

func TestDuplicateAndDeleteRequest(t *testing.T) {
	s := testStore(t)
	created, err := s.CreateRequest(Request{Name: "Original", Method: "GET", URL: "https://x.test"})
	if err != nil {
		t.Fatal(err)
	}

	copied, err := s.DuplicateRequest(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if copied.ID == created.ID || copied.Name != "Original copy" {
		t.Errorf("unexpected copy: %+v", copied)
	}

	if err := s.DeleteRequest(created.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteRequest(created.ID); err != ErrNotFound {
		t.Errorf("second delete should be ErrNotFound, got %v", err)
	}
}

func TestHistoryPrune(t *testing.T) {
	s := testStore(t)
	for i := 0; i < 5; i++ {
		if _, err := s.InsertHistory(HistoryEntry{Name: "run", Method: "GET", Status: 200, OK: true}); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.PruneHistory(2); err != nil {
		t.Fatal(err)
	}
	left, err := s.ListHistory("", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 2 {
		t.Errorf("kept %d entries, want 2", len(left))
	}

	n, err := s.ClearHistory()
	if err != nil || n != 2 {
		t.Errorf("clear = %d, %v", n, err)
	}
}

func TestEnvironmentAndActiveSetting(t *testing.T) {
	s := testStore(t)
	env, err := s.CreateEnvironment("Staging", []EnvVar{
		{Key: "baseUrl", Value: "https://staging.test", Enabled: true},
		{Key: "secret", Value: "s", Enabled: true, Secret: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(env.Variables) != 2 {
		t.Fatalf("variables = %+v", env.Variables)
	}
	if !env.Variables[1].Secret {
		t.Error("secret flag did not round-trip")
	}

	if err := s.SetSetting("activeEnvironmentId", env.ID); err != nil {
		t.Fatal(err)
	}
	if got := s.GetSetting("activeEnvironmentId", ""); got != env.ID {
		t.Errorf("setting = %q", got)
	}

	updated, err := s.UpdateEnvironment(env.ID, "Prod", []EnvVar{{Key: "baseUrl", Value: "https://prod.test", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Variables) != 1 || updated.Variables[0].Value != "https://prod.test" {
		t.Errorf("variables were not replaced: %+v", updated.Variables)
	}

	if err := s.DeleteEnvironment(env.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetEnvironment(env.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	s := testStore(t)
	col, _ := s.CreateCollection("Keep me")
	_, _ = s.CreateRequest(Request{CollectionID: &col.ID, Name: "Ping", Method: "GET", URL: "https://x.test"})
	_, _ = s.CreateEnvironment("Dev", []EnvVar{{Key: "k", Value: "v", Enabled: true}})

	snap, err := s.Snap()
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Requests) != 1 || len(snap.Collections) != 1 {
		t.Fatalf("snapshot = %+v", snap)
	}

	// Restore into a fresh database to prove the format is self-contained.
	other := testStore(t)
	if err := other.Restore(snap); err != nil {
		t.Fatal(err)
	}
	restored, err := other.ListRequests()
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 1 || restored[0].Name != "Ping" {
		t.Errorf("restored requests = %+v", restored)
	}
	envs, err := other.ListEnvironments()
	if err != nil || len(envs) != 1 || envs[0].Variables[0].Key != "k" {
		t.Errorf("restored environments = %+v, %v", envs, err)
	}
}
