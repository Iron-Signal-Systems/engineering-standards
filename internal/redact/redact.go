package redact

import "regexp"

var patterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)[^\s]+`),
	regexp.MustCompile(`(?i)((?:password|passwd|api[_-]?key|client[_-]?secret|access[_-]?token|refresh[_-]?token|secret|token)\s*[:=]\s*["']?)[^\s,"';]+`),
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),
}

func Sanitize(value string) string {
	out := value
	for _, pattern := range patterns {
		if pattern.NumSubexp() > 0 {
			out = pattern.ReplaceAllString(out, `${1}[REDACTED]`)
		} else {
			out = pattern.ReplaceAllString(out, `[REDACTED]`)
		}
	}
	return out
}
