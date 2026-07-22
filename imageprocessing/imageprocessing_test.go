package imageprocessing

import (
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	cases := []struct {
		ip      string
		private bool
	}{
		// Loopback / RFC1918 / link-local.
		{"127.0.0.1", true},
		{"10.1.2.3", true},
		{"172.16.5.5", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true}, // cloud metadata
		// New IPv4 ranges.
		{"100.64.0.1", true}, // CGNAT
		{"192.0.0.1", true},  // IETF protocol assignments
		{"192.0.2.5", true},  // TEST-NET-1
		{"198.18.0.1", true}, // benchmarking
		{"240.0.0.1", true},  // reserved / Class E
		{"255.255.255.255", true},
		{"0.0.0.0", true}, // unspecified
		// IPv4-mapped IPv6 must be unwrapped and caught.
		{"::ffff:169.254.169.254", true},
		{"::ffff:10.0.0.1", true},
		// IPv6 ranges.
		{"::1", true},        // loopback
		{"fc00::1", true},    // unique-local
		{"fe80::1", true},    // link-local
		{"ff02::1", true},    // multicast
		{"64:ff9b::1", true}, // NAT64
		// Public addresses must not be flagged.
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"93.184.216.34", false}, // example.com
		{"2606:2800:220:1:248:1893:25c8:1946", false},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("failed to parse test IP %q", c.ip)
		}
		if got := isPrivateIP(ip); got != c.private {
			t.Errorf("isPrivateIP(%s) = %v, want %v", c.ip, got, c.private)
		}
	}
}

func newTestService(hosts ...string) *Service {
	cfg := ServiceConfig{Format: "jpeg"}
	if len(hosts) > 0 {
		return New(nil, cfg, WithAllowedHosts(hosts))
	}
	return New(nil, cfg)
}

func TestFetchURLRejectsBadSchemeAndLocalhost(t *testing.T) {
	s := newTestService()
	cases := []string{
		"file:///etc/passwd",
		"gopher://evil/",
		"http://localhost/x",
		"https://localhost:443/x",
	}
	for _, u := range cases {
		if _, err := s.fetchURL(u); err == nil {
			t.Errorf("fetchURL(%q) expected error, got nil", u)
		}
	}
}

func TestFetchURLRejectsDisallowedPorts(t *testing.T) {
	s := newTestService()
	bad := []string{
		"http://example.com:22/x",
		"http://example.com:8080/x",
		"https://example.com:3306/x",
	}
	for _, u := range bad {
		if _, err := s.fetchURL(u); err == nil {
			t.Errorf("fetchURL(%q) expected port rejection, got nil", u)
		}
	}
}

func TestFetchURLAllowlist(t *testing.T) {
	s := newTestService("images.example.com")
	// Disallowed host should be rejected before any network activity.
	if _, err := s.fetchURL("http://evil.com/x"); err == nil {
		t.Errorf("fetchURL to non-allowlisted host expected error, got nil")
	}
	// Allowlist matching is case-insensitive.
	if _, ok := s.allowedHosts["images.example.com"]; !ok {
		t.Errorf("allowedHosts missing expected entry")
	}
}

func TestWithAllowedHostsNormalizes(t *testing.T) {
	s := newTestService("  IMAGES.Example.COM  ", "", "cdn.example.com")
	if len(s.allowedHosts) != 2 {
		t.Fatalf("expected 2 allowed hosts, got %d", len(s.allowedHosts))
	}
	if _, ok := s.allowedHosts["images.example.com"]; !ok {
		t.Errorf("host not lowercased/trimmed")
	}
}
