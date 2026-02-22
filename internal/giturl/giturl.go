package giturl

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// ParsedURL represents a normalized git repository URL.
type ParsedURL struct {
	Host  string // e.g. "github.com"
	Owner string // e.g. "anthropics" (or "group/subgroup" for GitLab)
	Repo  string // e.g. "claude-code"
}

// Parse handles HTTPS, SSH (git@host:owner/repo), ssh:// scheme, and .git suffix.
func Parse(rawURL string) (ParsedURL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ParsedURL{}, fmt.Errorf("empty URL")
	}

	var host, pathStr string

	// Handle SCP-style SSH: git@host:owner/repo
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, ":") && !strings.Contains(rawURL, "://") {
		parts := strings.SplitN(rawURL, "@", 2)
		hostAndPath := parts[1]
		colonIdx := strings.Index(hostAndPath, ":")
		if colonIdx < 0 {
			return ParsedURL{}, fmt.Errorf("invalid SCP-style URL: %s", rawURL)
		}
		host = hostAndPath[:colonIdx]
		pathStr = hostAndPath[colonIdx+1:]
	} else {
		// Ensure there's a scheme for url.Parse
		normalized := rawURL
		if !strings.Contains(normalized, "://") {
			normalized = "https://" + normalized
		}

		u, err := url.Parse(normalized)
		if err != nil {
			return ParsedURL{}, fmt.Errorf("invalid URL %q: %w", rawURL, err)
		}

		host = u.Hostname()
		pathStr = u.Path
	}

	if host == "" {
		return ParsedURL{}, fmt.Errorf("no host in URL: %s", rawURL)
	}

	// Clean up path
	pathStr = strings.TrimPrefix(pathStr, "/")
	pathStr = strings.TrimSuffix(pathStr, ".git")
	pathStr = strings.TrimSuffix(pathStr, "/")

	if pathStr == "" {
		return ParsedURL{}, fmt.Errorf("no path in URL: %s", rawURL)
	}

	// Split into owner and repo
	parts := strings.SplitN(pathStr, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return ParsedURL{}, fmt.Errorf("URL must have owner and repo: %s", rawURL)
	}

	// For GitLab-style nested groups, owner is everything before the last segment
	owner := path.Dir(pathStr)
	repo := path.Base(pathStr)

	if owner == "." {
		owner = parts[0]
	}

	return ParsedURL{
		Host:  host,
		Owner: owner,
		Repo:  repo,
	}, nil
}

// NormalizedURL returns the canonical HTTPS URL.
func (p ParsedURL) NormalizedURL() string {
	return fmt.Sprintf("https://%s/%s/%s", p.Host, p.Owner, p.Repo)
}

// CacheID returns a unique identifier for a repo+branch combination.
func (p ParsedURL) CacheID(branch string) string {
	return fmt.Sprintf("%s/%s/%s@%s", p.Host, p.Owner, p.Repo, branch)
}

// CachePath returns the relative filesystem path for caching this repo+branch.
func (p ParsedURL) CachePath(branch string) string {
	return fmt.Sprintf("repos/%s/%s/%s/%s", p.Host, p.Owner, p.Repo, branch)
}
