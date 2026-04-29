package main

import "testing"

func TestCountPublishersOnlyReady(t *testing.T) {
	pl := PathList{
		Items: []Path{
			{Name: "a", Ready: true},
			{Name: "b", Ready: false},
			{Name: "c", Ready: true},
		},
	}
	if got := countPublishers(pl); got != 2 {
		t.Errorf("countPublishers = %d, want 2", got)
	}
}

func TestCountReadersSumsAcrossPaths(t *testing.T) {
	pl := PathList{
		Items: []Path{
			{Name: "a", Readers: []any{"r1", "r2"}},
			{Name: "b", Readers: []any{"r3"}},
			{Name: "c", Readers: nil},
		},
	}
	if got := countReaders(pl); got != 3 {
		t.Errorf("countReaders = %d, want 3", got)
	}
}

func TestCountersOnEmptyList(t *testing.T) {
	pl := PathList{}
	if got := countPublishers(pl); got != 0 {
		t.Errorf("countPublishers(empty) = %d, want 0", got)
	}
	if got := countReaders(pl); got != 0 {
		t.Errorf("countReaders(empty) = %d, want 0", got)
	}
}
