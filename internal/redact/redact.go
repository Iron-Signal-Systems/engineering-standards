package redact

import (
	"bytes"
	"io"
	"regexp"
	"sync"
)

const maxPendingLine = 64 * 1024

var (
	privateKeyBlockPattern = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	privateKeyBeginPattern = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)
	privateKeyEndPattern   = regexp.MustCompile(`-----END [A-Z0-9 ]*PRIVATE KEY-----`)
)

var replacements = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{
		pattern:     privateKeyBlockPattern,
		replacement: `[REDACTED PRIVATE KEY]`,
	},
	{
		pattern:     regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)[^/@\s]+@`),
		replacement: `${1}[REDACTED]@`,
	},
	{
		pattern:     regexp.MustCompile(`(?i)(authorization\s*:\s*(?:bearer|basic)\s+)[^\s]+`),
		replacement: `${1}[REDACTED]`,
	},
	{
		pattern:     regexp.MustCompile(`(?i)((?:--?[a-z0-9_-]*(?:password|passwd|api[_-]?key|client[_-]?secret|access[_-]?token|refresh[_-]?token|secret|token)[a-z0-9_-]*)(?:=|\s+))(?:(?:"[^"\r\n]*")|(?:'[^'\r\n]*')|[^\s,;]+)`),
		replacement: `${1}[REDACTED]`,
	},
	{
		pattern:     regexp.MustCompile(`(?i)((?:[a-z0-9_-]*(?:password|passwd|api[_-]?key|client[_-]?secret|access[_-]?token|refresh[_-]?token|secret|token)[a-z0-9_-]*)\s*[:=]\s*)(?:(?:"[^"\r\n]*")|(?:'[^'\r\n]*')|[^\s,;]+)`),
		replacement: `${1}[REDACTED]`,
	},
	{
		pattern:     regexp.MustCompile(`github_pat_[A-Za-z0-9_]{20,}`),
		replacement: `[REDACTED]`,
	},
	{
		pattern:     regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}`),
		replacement: `[REDACTED]`,
	},
	{
		pattern:     regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		replacement: `[REDACTED]`,
	},
	{
		pattern:     regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),
		replacement: `[REDACTED]`,
	},
}

// Sanitize replaces credential-shaped values with explicit redaction markers.
// It is intended for terminal output, retained logs, and error text. It does not
// repair source and does not change command success or failure.
func Sanitize(value string) string {
	out := value
	for _, current := range replacements {
		out = current.pattern.ReplaceAllString(out, current.replacement)
	}
	return out
}

// Writer censors complete output lines before they reach the wrapped writer.
// Values split across multiple Write calls remain protected because an
// incomplete line is retained until a line boundary or Flush call. Multiline
// private-key blocks are replaced as one marker even when their lines arrive in
// separate writes. A line that exceeds the bounded pending-line budget is
// discarded rather than emitted without complete censoring context.
type Writer struct {
	mu                 sync.Mutex
	target             io.Writer
	pending            []byte
	droppingOversized  bool
	droppingPrivateKey bool
}

func NewWriter(target io.Writer) *Writer {
	if target == nil {
		target = io.Discard
	}
	return &Writer{target: target}
}

func (w *Writer) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	originalLength := len(data)
	for len(data) > 0 {
		if w.droppingOversized {
			boundary := bytes.IndexAny(data, "\r\n")
			if boundary < 0 {
				return originalLength, nil
			}
			if err := writeAll(w.target, []byte("[REDACTED: output line exceeded safe censoring limit]"+string(data[boundary]))); err != nil {
				return 0, err
			}
			w.droppingOversized = false
			data = data[boundary+1:]
			continue
		}

		boundary := bytes.IndexAny(data, "\r\n")
		if boundary >= 0 {
			line := make([]byte, 0, len(w.pending)+boundary+1)
			line = append(line, w.pending...)
			line = append(line, data[:boundary+1]...)
			w.pending = w.pending[:0]
			if err := w.writeLine(line); err != nil {
				return 0, err
			}
			data = data[boundary+1:]
			continue
		}

		if len(w.pending)+len(data) > maxPendingLine {
			w.pending = w.pending[:0]
			w.droppingOversized = true
			return originalLength, nil
		}
		w.pending = append(w.pending, data...)
		return originalLength, nil
	}
	return originalLength, nil
}

func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.droppingOversized {
		w.droppingOversized = false
		if err := writeAll(w.target, []byte("[REDACTED: output line exceeded safe censoring limit]")); err != nil {
			return err
		}
	}
	if len(w.pending) == 0 {
		return nil
	}
	line := append([]byte(nil), w.pending...)
	w.pending = w.pending[:0]
	return w.writeLine(line)
}

func (w *Writer) writeLine(line []byte) error {
	lineEnding := lineEnding(line)
	text := string(line)

	if w.droppingPrivateKey {
		if privateKeyEndPattern.MatchString(text) {
			w.droppingPrivateKey = false
		}
		return nil
	}

	if privateKeyBeginPattern.MatchString(text) && !privateKeyEndPattern.MatchString(text) {
		w.droppingPrivateKey = true
		return writeAll(w.target, []byte("[REDACTED PRIVATE KEY]"+lineEnding))
	}

	if len(line) > maxPendingLine {
		return writeAll(w.target, []byte("[REDACTED: output line exceeded safe censoring limit]"+lineEnding))
	}
	return writeAll(w.target, []byte(Sanitize(text)))
}

func lineEnding(line []byte) string {
	if len(line) == 0 {
		return ""
	}
	last := line[len(line)-1]
	if last == '\r' || last == '\n' {
		return string(last)
	}
	return ""
}

func writeAll(target io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := target.Write(data)
		if written > 0 {
			data = data[written:]
		}
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}
