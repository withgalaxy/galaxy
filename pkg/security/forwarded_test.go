package security

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/config"
)

func TestForwardedHostValidator_NoHeader(t *testing.T) {
	port := 443
	validator := NewForwardedHostValidator([]config.RemotePattern{
		{Protocol: "https", Hostname: "example.com", Port: &port},
	})

	req := httptest.NewRequest("GET", "https://original.com/path", nil)
	currentURL, _ := url.Parse("https://original.com/path")

	result := validator.ValidateForwardedHost(req, currentURL)

	if result.Host != "original.com" {
		t.Errorf("Expected original.com, got %s", result.Host)
	}
}

func TestForwardedHostValidator_NoPatterns(t *testing.T) {
	validator := NewForwardedHostValidator([]config.RemotePattern{})

	req := httptest.NewRequest("GET", "https://original.com/path", nil)
	req.Header.Set("X-Forwarded-Host", "evil.com")
	currentURL, _ := url.Parse("https://original.com/path")

	result := validator.ValidateForwardedHost(req, currentURL)

	if result.Host != "original.com" {
		t.Errorf("Expected original.com (rejected), got %s", result.Host)
	}
}

func TestForwardedHostValidator_MatchesPattern(t *testing.T) {
	validator := NewForwardedHostValidator([]config.RemotePattern{
		{Protocol: "https", Hostname: "example.com"},
	})

	req := httptest.NewRequest("GET", "https://original.com/path", nil)
	req.Header.Set("X-Forwarded-Host", "example.com")
	currentURL, _ := url.Parse("https://original.com/path")

	result := validator.ValidateForwardedHost(req, currentURL)

	if result.Host != "example.com" {
		t.Errorf("Expected example.com, got %s", result.Host)
	}
	if result.Scheme != "https" {
		t.Errorf("Expected https, got %s", result.Scheme)
	}
}

func TestForwardedHostValidator_WildcardMatching(t *testing.T) {
	validator := NewForwardedHostValidator([]config.RemotePattern{
		{Protocol: "https", Hostname: "*.example.com"},
	})

	tests := []struct {
		name        string
		host        string
		shouldMatch bool
	}{
		{"subdomain", "api.example.com", true},
		{"nested", "foo.bar.example.com", true},
		{"base", "example.com", true},
		{"nomatch", "notexample.com", false},
		{"nomatch2", "examplecom.evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "https://original.com/path", nil)
			req.Header.Set("X-Forwarded-Host", tt.host)
			currentURL, _ := url.Parse("https://original.com/path")

			result := validator.ValidateForwardedHost(req, currentURL)

			if tt.shouldMatch {
				if result.Host != tt.host {
					t.Errorf("Expected %s, got %s", tt.host, result.Host)
				}
			} else {
				if result.Host != "original.com" {
					t.Errorf("Expected original.com (rejected), got %s", result.Host)
				}
			}
		})
	}
}

func TestForwardedHostValidator_PortMatching(t *testing.T) {
	port8080 := 8080
	validator := NewForwardedHostValidator([]config.RemotePattern{
		{Protocol: "https", Hostname: "example.com", Port: &port8080},
	})

	tests := []struct {
		name        string
		host        string
		shouldMatch bool
	}{
		{"correct port", "example.com:8080", true},
		{"wrong port", "example.com:9000", false},
		{"no port", "example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "https://original.com/path", nil)
			req.Header.Set("X-Forwarded-Host", tt.host)
			currentURL, _ := url.Parse("https://original.com/path")

			result := validator.ValidateForwardedHost(req, currentURL)

			if tt.shouldMatch {
				if result.Host != tt.host {
					t.Errorf("Expected %s, got %s", tt.host, result.Host)
				}
			} else {
				if result.Host != "original.com" {
					t.Errorf("Expected original.com (rejected), got %s", result.Host)
				}
			}
		})
	}
}

func TestForwardedHostValidator_Sanitization(t *testing.T) {
	port := 443
	validator := NewForwardedHostValidator([]config.RemotePattern{
		{Protocol: "https", Hostname: "example.com", Port: &port},
	})

	tests := []struct {
		name  string
		value string
	}{
		{"path injection", "example.com/evil"},
		{"empty", ""},
		{"whitespace", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "https://original.com/path", nil)
			req.Header.Set("X-Forwarded-Host", tt.value)
			currentURL, _ := url.Parse("https://original.com/path")

			result := validator.ValidateForwardedHost(req, currentURL)

			if result.Host != "original.com" {
				t.Errorf("Expected original.com (rejected malicious), got %s", result.Host)
			}
		})
	}
}
