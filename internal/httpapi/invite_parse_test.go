package httpapi

import (
	"strings"
	"testing"
)

func TestParseInvitationAcceptsCommonForms(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		host string
		code string
	}{
		{"full url", "http://192.168.64.1:47810/join/abc123", "192.168.64.1:47810", "abc123"},
		{"trailing slash", "http://10.0.0.4:47810/join/xyz789/", "10.0.0.4:47810", "xyz789"},
		{"no scheme", "192.168.64.1:47810/join/noscheme", "192.168.64.1:47810", "noscheme"},
		{"hostname", "http://macbook.local:47810/join/hn", "macbook.local:47810", "hn"},
		{"uppercase join", "http://192.168.64.1:47810/JOIN/case", "192.168.64.1:47810", "case"},
		{"wrapped in angle brackets", "<http://192.168.64.1:47810/join/wrap>", "192.168.64.1:47810", "wrap"},
		{"trailing space and newline", "http://192.168.64.1:47810/join/space \n", "192.168.64.1:47810", "space"},
		{"extra path segments", "http://192.168.64.1:47810/api/something/join/deep", "192.168.64.1:47810", "deep"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, code, err := parseInvitation(tc.raw)
			if err != nil {
				t.Fatalf("parseInvitation(%q) error: %v", tc.raw, err)
			}
			if parsed.Host != tc.host {
				t.Fatalf("host = %q, want %q", parsed.Host, tc.host)
			}
			if code != tc.code {
				t.Fatalf("code = %q, want %q", code, tc.code)
			}
			if !strings.HasPrefix(parsed.Scheme, "http") {
				t.Fatalf("unexpected scheme %q", parsed.Scheme)
			}
		})
	}
}

func TestParseInvitationRejectsInvalid(t *testing.T) {
	cases := []string{"", "   ", "not a url", "http://192.168.64.1/connect", "http://192.168.64.1/join/"}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			_, _, err := parseInvitation(raw)
			if err == nil {
				t.Fatalf("parseInvitation(%q) expected error", raw)
			}
		})
	}
}
