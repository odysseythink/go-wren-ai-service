package provider

import "testing"

func TestRegistry(t *testing.T) {
	Register("test", func(cfg map[string]any) (any, error) { return "ok", nil })
	f, ok := Get("test")
	if !ok {
		t.Fatal("expected provider to be registered")
	}
	result, err := f(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestRegistryNotFound(t *testing.T) {
	_, ok := Get("nonexistent")
	if ok {
		t.Fatal("expected provider not to be registered")
	}
}
