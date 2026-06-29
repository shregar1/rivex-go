package ffi

import "testing"

func TestVersion(t *testing.T) {
	v := Version()
	if v == "" {
		t.Fatal("expected non-empty version string from rivex-ffi")
	}
	if v != "0.1.0" {
		t.Fatalf("expected version 0.1.0, got %s", v)
	}
	t.Logf("rivex-ffi version: %s", v)
}

func TestServerNewAndFree(t *testing.T) {
	server := ServerNew()
	if server == nil {
		t.Fatal("ServerNew returned nil")
	}
	ServerFree(server)
}
