package executil

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

type Result struct {
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
	Started  time.Time
	Finished time.Time
	Err      error
}

func Run(ctx context.Context, dir, name string, args ...string) Result {
	started := time.Now()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitCode = -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	return Result{
		Command:  commandString(name, args),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Started:  started,
		Finished: time.Now(),
		Err:      err,
	}
}

func commandString(name string, args []string) string {
	value := name
	for _, arg := range args {
		value += " " + shellQuote(arg)
	}
	return value
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	safe := true
	for _, r := range value {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '/' && r != '.' && r != '_' && r != '-' && r != ':' && r != '@' {
			safe = false
			break
		}
	}
	if safe {
		return value
	}
	out := "'"
	for _, r := range value {
		if r == '\'' {
			out += "'\\''"
		} else {
			out += string(r)
		}
	}
	return out + "'"
}
