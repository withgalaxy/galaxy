package security

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/middleware"
)

type CSRFMiddleware struct {
	config         *CSRFConfig
	allowedOrigins map[string]bool
}

func NewCSRFMiddleware(cfg *CSRFConfig) *CSRFMiddleware {
	return &CSRFMiddleware{
		config:         cfg,
		allowedOrigins: GetAllowedOrigins(cfg.SiteURL, cfg.AllowOrigins),
	}
}

func (m *CSRFMiddleware) Middleware(ctx *middleware.Context, next func() error) error {
	if !m.config.CheckOrigin {
		return next()
	}

	if !shouldCheckCSRF(ctx.Request) {
		return next()
	}

	origin := GetOriginFromRequest(ctx.Request)

	if origin == "" {
		http.Error(ctx.Response, "Forbidden: Origin header required", http.StatusForbidden)
		return fmt.Errorf("origin header missing")
	}

	if !m.isOriginAllowed(origin) {
		http.Error(ctx.Response, "Forbidden: Origin not allowed", http.StatusForbidden)
		return fmt.Errorf("origin not allowed: %s", origin)
	}

	return next()
}

func (m *CSRFMiddleware) isOriginAllowed(origin string) bool {
	if IsLocalhost(origin) {
		return true
	}

	normalized := NormalizeOrigin(origin)
	return m.allowedOrigins[normalized]
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet ||
		method == http.MethodHead ||
		method == http.MethodOptions
}

func shouldCheckCSRF(r *http.Request) bool {
	if isSafeMethod(r.Method) {
		return false
	}

	contentType := r.Header.Get("Content-Type")
	ct := strings.Split(contentType, ";")[0]
	ct = strings.TrimSpace(ct)

	return ct == "application/x-www-form-urlencoded" ||
		ct == "multipart/form-data" ||
		ct == "text/plain" ||
		ct == ""
}
