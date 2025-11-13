package security

import (
	"net/http"
	"net/url"
	"strings"
)

func GetOriginFromRequest(r *http.Request) string {
	if origin := r.Header.Get("Origin"); origin != "" {
		return NormalizeOrigin(origin)
	}

	if referer := r.Header.Get("Referer"); referer != "" {
		if u, err := url.Parse(referer); err == nil {
			return NormalizeOrigin(u.Scheme + "://" + u.Host)
		}
	}

	return ""
}

func NormalizeOrigin(origin string) string {
	origin = strings.ToLower(origin)
	origin = strings.TrimSuffix(origin, "/")
	return origin
}

func IsLocalhost(origin string) bool {
	return strings.Contains(origin, "localhost") ||
		strings.Contains(origin, "127.0.0.1") ||
		strings.Contains(origin, "[::1]")
}

func GetAllowedOrigins(siteURL string, additional []string) map[string]bool {
	allowed := make(map[string]bool)

	if siteURL != "" {
		allowed[NormalizeOrigin(siteURL)] = true
	}

	for _, origin := range additional {
		if origin != "" {
			allowed[NormalizeOrigin(origin)] = true
		}
	}

	return allowed
}
