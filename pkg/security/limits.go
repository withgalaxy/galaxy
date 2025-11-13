package security

import (
	"net/http"

	"github.com/withgalaxy/galaxy/pkg/middleware"
)

type BodyLimitMiddleware struct {
	maxBytes int64
}

func NewBodyLimitMiddleware(maxBytes int64) *BodyLimitMiddleware {
	return &BodyLimitMiddleware{maxBytes: maxBytes}
}

func (m *BodyLimitMiddleware) Middleware(ctx *middleware.Context, next func() error) error {
	ctx.Request.Body = http.MaxBytesReader(ctx.Response, ctx.Request.Body, m.maxBytes)
	return next()
}
