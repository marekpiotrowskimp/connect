package idgen

import "testing"

func TestNewIsUnique(t *testing.T) {
	a := New()
	b := New()
	if a == b {
		t.Fatal("expected two calls to produce different ids")
	}
	if len(a) != 36 {
		t.Fatalf("expected uuid-formatted id of length 36, got %d (%q)", len(a), a)
	}
}
