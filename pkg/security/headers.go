package security

import (
	"github.com/withgalaxy/galaxy/pkg/config"
	"github.com/withgalaxy/galaxy/pkg/middleware"
)

type HeadersMiddleware struct {
	config config.HeadersConfig
}

func NewHeadersMiddleware(cfg config.HeadersConfig) *HeadersMiddleware {
	return &HeadersMiddleware{config: cfg}
}

func (m *HeadersMiddleware) Middleware(ctx *middleware.Context, next func() error) error {
	if !m.config.Enabled {
		return next()
	}

	if m.config.XFrameOptions != "" {
		ctx.Response.Header().Set("X-Frame-Options", m.config.XFrameOptions)
	}
	if m.config.XContentTypeOptions != "" {
		ctx.Response.Header().Set("X-Content-Type-Options", m.config.XContentTypeOptions)
	}
	if m.config.XXSSProtection != "" {
		ctx.Response.Header().Set("X-XSS-Protection", m.config.XXSSProtection)
	}
	if m.config.ReferrerPolicy != "" {
		ctx.Response.Header().Set("Referrer-Policy", m.config.ReferrerPolicy)
	}
	if m.config.StrictTransportSecurity != "" {
		ctx.Response.Header().Set("Strict-Transport-Security", m.config.StrictTransportSecurity)
	}
	if m.config.ContentSecurityPolicy != "" {
		ctx.Response.Header().Set("Content-Security-Policy", m.config.ContentSecurityPolicy)
	}
	if m.config.PermissionsPolicy != "" {
		ctx.Response.Header().Set("Permissions-Policy", m.config.PermissionsPolicy)
	}

	return next()
}
