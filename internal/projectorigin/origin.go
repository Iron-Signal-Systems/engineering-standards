package projectorigin

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var repositoryNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)

const Organization = "Iron-Signal-Systems"

// Canonical validates one supported Iron Signal Systems GitHub origin and
// returns the stable repository identity committed in project pins.
func Canonical(origin string) (string, error) {
	origin = strings.TrimSpace(origin)
	if origin == "" || strings.ContainsAny(origin, "\x00\r\n\t") {
		return "", errors.New("target repository origin is unavailable or invalid")
	}

	var repositoryPath string
	if strings.HasPrefix(origin, "git@github.com:") {
		repositoryPath = strings.TrimPrefix(origin, "git@github.com:")
		if strings.Contains(repositoryPath, ":") {
			return "", errors.New("target repository origin contains an unexpected port or path separator")
		}
	} else {
		parsed, err := url.Parse(origin)
		if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") || parsed.Port() != "" {
			return "", errors.New("target repository origin is not a canonical GitHub URL")
		}
		if parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" {
			return "", errors.New("target repository origin contains unsupported URL components")
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "ssh" && scheme != "https" {
			return "", errors.New("target repository origin uses an unsupported transport")
		}
		if parsed.User != nil {
			_, hasPassword := parsed.User.Password()
			if scheme != "ssh" || parsed.User.Username() != "git" || hasPassword {
				return "", errors.New("target repository origin contains unsupported credentials")
			}
		}
		if scheme == "ssh" && parsed.User == nil {
			return "", errors.New("target repository SSH origin requires the git user")
		}
		repositoryPath = strings.TrimPrefix(parsed.Path, "/")
	}

	repositoryPath = strings.TrimSuffix(repositoryPath, ".git")
	parts := strings.Split(repositoryPath, "/")
	if len(parts) != 2 || parts[0] != Organization || !repositoryNamePattern.MatchString(parts[1]) {
		return "", errors.New("target repository origin has an unexpected repository identity")
	}
	return "github.com/" + Organization + "/" + parts[1], nil
}
