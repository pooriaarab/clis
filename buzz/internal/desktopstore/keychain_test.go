package desktopstore

import "testing"

func TestMemorySecretStoreRoundTrip(t *testing.T) {
	store := NewMemorySecretStore()

	if err := store.Store("agent:one", "nsec-one"); err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	got, ok, err := store.Load("agent:one")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !ok || got != "nsec-one" {
		t.Fatalf("Load() = %q, %v, want nsec-one, true", got, ok)
	}

	got, ok, err = store.Load("agent:missing")
	if err != nil {
		t.Fatalf("Load(missing) error = %v", err)
	}
	if ok || got != "" {
		t.Fatalf("Load(missing) = %q, %v, want empty, false", got, ok)
	}

	if err := store.Delete("agent:one"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	got, ok, err = store.Load("agent:one")
	if err != nil {
		t.Fatalf("Load(after delete) error = %v", err)
	}
	if ok || got != "" {
		t.Fatalf("Load(after delete) = %q, %v, want empty, false", got, ok)
	}

	if err := store.Delete("agent:absent"); err != nil {
		t.Fatalf("Delete(absent) error = %v", err)
	}
}

func TestNewDarwinKeychainStoreDoesNotPanic(t *testing.T) {
	store := NewDarwinKeychainStore("buzz-desktop-cli-test-DO-NOT-USE")
	if store == nil {
		t.Fatal("NewDarwinKeychainStore() returned nil")
	}
}
