package failurelog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/executil"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/redact"
)

type Context struct {
	Root     string
	Mode     string
	Check    string
	Expected string
	Observed string
	Actions  []string
	Command  *executil.Result
	Commit   string
	Branch   string
	Started  time.Time
	Finished time.Time
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func Write(ctx Context) (string, error) {
	dir := filepath.Join(ctx.Root, ".local", "validation", "logs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	slug := strings.Trim(slugPattern.ReplaceAllString(strings.ToLower(ctx.Check), "-"), "-")
	if slug == "" {
		slug = "validation"
	}
	path := filepath.Join(dir, stamp+"-"+slug+".log")
	var b strings.Builder
	b.WriteString("Iron Signal Validation Failure\n")
	b.WriteString("================================\n\n")
	writeField(&b, "Repository root", ctx.Root)
	writeField(&b, "Mode", ctx.Mode)
	writeField(&b, "Check", ctx.Check)
	writeField(&b, "Branch", ctx.Branch)
	writeField(&b, "Commit", ctx.Commit)
	if !ctx.Started.IsZero() {
		writeField(&b, "Started", ctx.Started.UTC().Format(time.RFC3339))
	}
	if !ctx.Finished.IsZero() {
		writeField(&b, "Finished", ctx.Finished.UTC().Format(time.RFC3339))
	}
	writeField(&b, "Expected", ctx.Expected)
	writeField(&b, "Observed", ctx.Observed)
	if ctx.Command != nil {
		writeField(&b, "Command", ctx.Command.Command)
		writeField(&b, "Exit code", fmt.Sprintf("%d", ctx.Command.ExitCode))
		b.WriteString("\nStandard output\n---------------\n")
		b.WriteString(clean(ctx.Command.Stdout))
		b.WriteString("\n\nStandard error\n--------------\n")
		b.WriteString(clean(ctx.Command.Stderr))
		b.WriteString("\n")
	}
	if len(ctx.Actions) > 0 {
		b.WriteString("\nAvailable actions\n-----------------\n")
		for _, action := range ctx.Actions {
			b.WriteString("- ")
			b.WriteString(redact.Sanitize(action))
			b.WriteString("\n")
		}
	}
	content := redact.Sanitize(b.String())
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return filepath.ToSlash(path), nil
}

func writeField(b *strings.Builder, label, value string) {
	if value == "" {
		value = "unavailable"
	}
	fmt.Fprintf(b, "%-16s %s\n", label+":", redact.Sanitize(value))
}

func clean(value string) string {
	value = redact.Sanitize(value)
	value = strings.TrimSpace(value)
	if value == "" {
		return "(no output)"
	}
	const max = 128 * 1024
	if len(value) > max {
		value = value[:max] + "\n[output truncated]"
	}
	return value
}
