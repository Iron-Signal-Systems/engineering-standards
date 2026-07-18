package projectorigin

import "testing"

func TestCanonical(t *testing.T) {
	for _, test := range []struct {
		origin string
		want   string
	}{
		{"git@github.com:Iron-Signal-Systems/iron-atlas.git", "github.com/Iron-Signal-Systems/iron-atlas"},
		{"https://github.com/Iron-Signal-Systems/iron-atlas.git", "github.com/Iron-Signal-Systems/iron-atlas"},
		{"ssh://git@github.com/Iron-Signal-Systems/iron-atlas.git", "github.com/Iron-Signal-Systems/iron-atlas"},
	} {
		got, err := Canonical(test.origin)
		if err != nil {
			t.Fatalf("Canonical(%q): %v", test.origin, err)
		}
		if got != test.want {
			t.Fatalf("Canonical(%q) = %q, want %q", test.origin, got, test.want)
		}
	}
}

func TestCanonicalRejectsUnsafeOrUnsupportedOrigins(t *testing.T) {
	for _, origin := range []string{
		"",
		"git://github.com/Iron-Signal-Systems/iron-atlas.git",
		"git@gitlab.com:Iron-Signal-Systems/iron-atlas.git",
		"git@github.com:Other/iron-atlas.git",
		"https://github.com:443/Iron-Signal-Systems/iron-atlas.git",
		"https://token@github.com/Iron-Signal-Systems/iron-atlas.git",
		"ssh://github.com/Iron-Signal-Systems/iron-atlas.git",
		"file://github.com/Iron-Signal-Systems/iron-atlas.git",
		"https://github.com/Iron-Signal-Systems/../iron-atlas.git",
		"https://github.com/Iron-Signal-Systems/iron-atlas.git?ref=dev",
		"https://github.com/Iron-Signal-Systems/iron-atlas.git#fragment",
	} {
		if _, err := Canonical(origin); err == nil {
			t.Fatalf("unsafe origin %q was accepted", origin)
		}
	}
}
