package releaseartifactbuild

import "testing"

func TestParseGoDirectiveRequiresPatchVersion(t *testing.T) {
	if got, err := parseGoDirective("module example.com/test\n\ngo 1.25.12\n"); err != nil || got != "1.25.12" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, value := range []string{
		"module example.com/test\n",
		"module example.com/test\n\ngo 1.25\n",
		"module example.com/test\n\ngo latest\n",
	} {
		if _, err := parseGoDirective(value); err == nil {
			t.Fatalf("expected invalid go.mod: %q", value)
		}
	}
}

func TestCanonicalOrigin(t *testing.T) {
	for _, value := range []string{
		"git@github.com:Iron-Signal-Systems/engineering-standards.git",
		"https://github.com/Iron-Signal-Systems/engineering-standards.git",
		"ssh://git@github.com/Iron-Signal-Systems/engineering-standards.git",
	} {
		if !canonicalOrigin(value) {
			t.Fatalf("canonical origin rejected: %s", value)
		}
	}
	for _, value := range []string{
		"https://github.com/Iron-Signal-Systems/engineering-standards",
		"git@github.com:someone/engineering-standards.git",
		"file:///tmp/repo",
	} {
		if canonicalOrigin(value) {
			t.Fatalf("noncanonical origin accepted: %s", value)
		}
	}
}

func TestGoVersionAtLeastUsesMinimumSemantics(t *testing.T) {
	tests := []struct {
		name    string
		actual  string
		minimum string
		want    bool
	}{
		{
			name:    "exact minimum",
			actual:  "go1.25.12",
			minimum: "go1.25.12",
			want:    true,
		},
		{
			name:    "later patch",
			actual:  "go1.25.13",
			minimum: "go1.25.12",
			want:    true,
		},
		{
			name:    "later custom toolchain",
			actual:  "go1.26.5-X:nodwarf5",
			minimum: "go1.25.12",
			want:    true,
		},
		{
			name:    "below minimum",
			actual:  "go1.25.11",
			minimum: "go1.25.12",
			want:    false,
		},
		{
			name:    "invalid actual",
			actual:  "1.26.5",
			minimum: "go1.25.12",
			want:    false,
		},
		{
			name:    "invalid minimum",
			actual:  "go1.26.5",
			minimum: "1.25.12",
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := goVersionAtLeast(
				test.actual,
				test.minimum,
			); got != test.want {
				t.Fatalf(
					"goVersionAtLeast(%q, %q) = %v, want %v",
					test.actual,
					test.minimum,
					got,
					test.want,
				)
			}
		})
	}
}
