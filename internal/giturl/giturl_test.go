package giturl

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ParsedURL
		wantErr bool
	}{
		{
			name:  "HTTPS basic",
			input: "https://github.com/owner/repo",
			want:  ParsedURL{Host: "github.com", Owner: "owner", Repo: "repo"},
		},
		{
			name:  "HTTPS with .git suffix",
			input: "https://github.com/owner/repo.git",
			want:  ParsedURL{Host: "github.com", Owner: "owner", Repo: "repo"},
		},
		{
			name:  "SSH SCP-style",
			input: "git@github.com:owner/repo.git",
			want:  ParsedURL{Host: "github.com", Owner: "owner", Repo: "repo"},
		},
		{
			name:  "SSH SCP-style without .git",
			input: "git@github.com:owner/repo",
			want:  ParsedURL{Host: "github.com", Owner: "owner", Repo: "repo"},
		},
		{
			name:  "ssh:// scheme",
			input: "ssh://git@github.com/owner/repo.git",
			want:  ParsedURL{Host: "github.com", Owner: "owner", Repo: "repo"},
		},
		{
			name:  "http:// upgraded to https",
			input: "http://github.com/owner/repo",
			want:  ParsedURL{Host: "github.com", Owner: "owner", Repo: "repo"},
		},
		{
			name:  "GitLab nested group",
			input: "https://gitlab.com/group/subgroup/repo",
			want:  ParsedURL{Host: "gitlab.com", Owner: "group/subgroup", Repo: "repo"},
		},
		{
			name:  "trailing slash",
			input: "https://github.com/owner/repo/",
			want:  ParsedURL{Host: "github.com", Owner: "owner", Repo: "repo"},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "host only",
			input:   "https://github.com",
			wantErr: true,
		},
		{
			name:    "host and owner only",
			input:   "https://github.com/owner",
			wantErr: true,
		},
		{
			name:    "garbage",
			input:   "not a url at all!!!",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSSHAndHTTPSEquivalence(t *testing.T) {
	ssh, err := Parse("git@github.com:anthropics/claude-code.git")
	if err != nil {
		t.Fatal(err)
	}
	https, err := Parse("https://github.com/anthropics/claude-code")
	if err != nil {
		t.Fatal(err)
	}
	if ssh != https {
		t.Errorf("SSH %+v != HTTPS %+v", ssh, https)
	}
}

func TestNormalizedURL(t *testing.T) {
	p := ParsedURL{Host: "github.com", Owner: "owner", Repo: "repo"}
	want := "https://github.com/owner/repo"
	if got := p.NormalizedURL(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCacheID(t *testing.T) {
	p := ParsedURL{Host: "github.com", Owner: "owner", Repo: "repo"}
	want := "github.com/owner/repo@main"
	if got := p.CacheID("main"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCachePath(t *testing.T) {
	p := ParsedURL{Host: "github.com", Owner: "owner", Repo: "repo"}
	want := "repos/github.com/owner/repo/main"
	if got := p.CachePath("main"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
