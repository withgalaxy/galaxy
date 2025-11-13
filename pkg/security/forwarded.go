package security

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/config"
)

type ForwardedHostValidator struct {
	allowedDomains []config.RemotePattern
}

func NewForwardedHostValidator(patterns []config.RemotePattern) *ForwardedHostValidator {
	return &ForwardedHostValidator{allowedDomains: patterns}
}

func (v *ForwardedHostValidator) ValidateForwardedHost(r *http.Request, currentURL *url.URL) *url.URL {
	forwardedHost := r.Header.Get("X-Forwarded-Host")
	if forwardedHost == "" {
		return currentURL
	}

	if len(v.allowedDomains) == 0 {
		return currentURL
	}

	sanitized := strings.TrimSpace(forwardedHost)
	if sanitized == "" || strings.Contains(sanitized, "/") {
		return currentURL
	}

	host, portStr, hasPort := strings.Cut(sanitized, ":")
	port := 0
	if hasPort {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return currentURL
		}
		port = p
	}

	for _, pattern := range v.allowedDomains {
		if matchesPattern(host, port, pattern) {
			newURL := *currentURL
			newURL.Host = sanitized
			if pattern.Protocol != "" {
				newURL.Scheme = pattern.Protocol
			}
			return &newURL
		}
	}

	return currentURL
}

func matchesPattern(host string, port int, pattern config.RemotePattern) bool {
	if !matchesHostname(host, pattern.Hostname) {
		return false
	}

	if pattern.Port != nil {
		if port != *pattern.Port {
			return false
		}
	}

	return true
}

func matchesHostname(host, pattern string) bool {
	host = strings.ToLower(host)
	pattern = strings.ToLower(pattern)

	if pattern == host {
		return true
	}

	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		return strings.HasSuffix(host, "."+suffix) || host == suffix
	}

	return false
}
