package uuid

import (
	"testing"
)

func TestUUIDUnique(t *testing.T) {
	u := NewUUID()
	if u.ID() == u.ID() {
		t.Error("UUIDs are not unique")
	}
}

func TestStaticIDs(t *testing.T) {
	u := NewStaticIDs("A", "B")
	for _, expected := range []string{"A", "B", "A", "B", "A"} {
		if have, want := u.ID(), expected; have != want {
			t.Errorf("unexpected ID: have: %v, want: %v", have, want)
		}
	}
}
