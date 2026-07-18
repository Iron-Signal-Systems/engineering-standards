package main

import "testing"

func TestParseProjectInitializationArgs(t *testing.T) {
	options, err := parseProjectInitializationArgs([]string{
		"--release", "isras-v0.1.2",
		"--go-defaults",
		"--evidence-directory", ".local/validation",
	})
	if err != nil {
		t.Fatalf("parse initialization arguments: %v", err)
	}
	if options.ReleaseTag != "isras-v0.1.2" || !options.GoDefaults || options.EvidenceDirectory != ".local/validation" {
		t.Fatalf("unexpected options: %#v", options)
	}
}

func TestParseProjectInitializationArgsRejectsIncompleteOrAmbiguousInput(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"--release", "isras-v0.1.2"},
		{"--go-defaults"},
		{"--release", "isras-v0.1.2", "--release", "isras-v0.1.3", "--go-defaults"},
		{"--release", "isras-v0.1.2", "--go-defaults", "--unknown"},
	} {
		if _, err := parseProjectInitializationArgs(args); err == nil {
			t.Fatalf("invalid arguments were accepted: %#v", args)
		}
	}
}
