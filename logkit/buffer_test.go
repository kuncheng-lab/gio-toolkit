package logkit

import "testing"

func TestAddAppendsAndIncrementsRevision(t *testing.T) {
	b := NewBuffer(10)
	b.Add("a")
	b.Addf("count=%d", 2)

	if b.Revision != 2 {
		t.Fatalf("revision = %d, want 2", b.Revision)
	}
	if len(b.Lines) != 2 || b.Lines[1] != "count=2" {
		t.Fatalf("lines = %v", b.Lines)
	}
}

func TestAddTrimsOldest(t *testing.T) {
	b := NewBuffer(3)
	for i := 0; i < 5; i++ {
		b.Addf("%d", i)
	}

	if len(b.Lines) != 3 {
		t.Fatalf("len = %d, want 3", len(b.Lines))
	}
	if b.Lines[0] != "2" || b.Lines[2] != "4" {
		t.Fatalf("lines = %v, want [2 3 4]", b.Lines)
	}
}

func TestCloneIsIndependent(t *testing.T) {
	b := NewBuffer(10)
	b.Add("a")

	clone := b.Clone()
	b.Add("b")

	if len(clone.Lines) != 1 {
		t.Fatalf("clone mutated: %v", clone.Lines)
	}
	if clone.Revision != 1 {
		t.Fatalf("clone revision = %d, want 1", clone.Revision)
	}
}

func TestTail(t *testing.T) {
	b := NewBuffer(10)
	for i := 0; i < 5; i++ {
		b.Addf("%d", i)
	}

	tail := b.Tail(2)
	if len(tail) != 2 || tail[0] != "3" || tail[1] != "4" {
		t.Fatalf("tail = %v, want [3 4]", tail)
	}
	if full := b.Tail(10); len(full) != 5 {
		t.Fatalf("full tail = %v", full)
	}
}
