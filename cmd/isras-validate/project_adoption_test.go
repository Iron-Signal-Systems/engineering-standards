package main

import "testing"

func TestParseProjectInitializationArgs(t *testing.T) {
	options, err := parseProjectInitializationArgs([]string{
		"--release", "isras-v0.1.2",
		"--go-defaults",
	})
	if err != nil {
		t.Fatalf("parse initialization arguments: %v", err)
	}
	if options.ReleaseTag != "isras-v0.1.2" || !options.GoDefaults {
		t.Fatalf("unexpected options: %#v", options)
	}
}

func TestParseProjectInitializationArgsRejectsUnsafeOrObsoleteOptions(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"--release", "isras-v0.1.2"},
		{"--go-defaults"},
		{"--release", "isras-v0.1.2", "--go-defaults", "--evidence-directory", ".isras"},
		{"--release", "isras-v0.1.2", "--release", "isras-v0.1.2", "--go-defaults"},
		{"--release", "isras-v0.1.2", "--go-defaults", "--go-defaults"},
	} {
		if _, err := parseProjectInitializationArgs(args); err == nil {
			t.Fatalf("unsafe arguments %#v were accepted", args)
		}
	}
}
