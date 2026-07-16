package repository

import "testing"

func TestLikePrefixPattern_PlainInput(t *testing.T) {
	got := likePrefixPattern("слив")
	want := "слив%"
	if got != want {
		t.Errorf("likePrefixPattern(%q) = %q, want %q", "слив", got, want)
	}
}

func TestLikePrefixPattern_EscapesWildcards(t *testing.T) {
	got := likePrefixPattern("50%_off")
	want := `50\%\_off%`
	if got != want {
		t.Errorf("likePrefixPattern(%q) = %q, want %q", "50%_off", got, want)
	}
}

func TestLikePrefixPattern_EscapesBackslash(t *testing.T) {
	got := likePrefixPattern(`a\b`)
	want := `a\\b%`
	if got != want {
		t.Errorf("likePrefixPattern(%q) = %q, want %q", `a\b`, got, want)
	}
}
