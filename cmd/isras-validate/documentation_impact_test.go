package main

import (
	"strings"
	"testing"
)

func TestParseDocumentationImpactOptions(t *testing.T) {
	base := strings.Repeat("a", 40)
	head := strings.Repeat("b", 40)

	options, err := parseDocumentationImpactOptions(
		[]string{"--head=" + head, "--base", base},
	)
	if err != nil {
		t.Fatal(err)
	}
	if options.BaseCommit != base || options.HeadCommit != head {
		t.Fatalf("options = %#v", options)
	}

	for _, args := range [][]string{
		{},
		{"--base", base},
		{"--head", head},
		{"--base", base, "--base", base, "--head", head},
		{"--base", base, "--head", head, "extra"},
	} {
		if _, err := parseDocumentationImpactOptions(args); err == nil {
			t.Fatalf("arguments accepted: %v", args)
		}
	}
}
