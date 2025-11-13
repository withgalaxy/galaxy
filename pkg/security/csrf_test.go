package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/middleware"
)

func TestCSRFMiddleware_SafeMethods(t *testing.T) {
	cfg := &CSRFConfig{
		CheckOrigin:  true,
		AllowOrigins: []string{},
		SiteURL:      "https://example.com",
	}
	csrfMw := NewCSRFMiddleware(cfg)

	safeMethods := []string{"GET", "HEAD", "OPTIONS"}

	for _, method := range safeMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()
			ctx := middleware.NewContext(w, req)

			called := false
			err := csrfMw.Middleware(ctx, func() error {
				called = true
				return nil
			})

			if err != nil {
				t.Errorf("Expected no error for %s, got %v", method, err)
			}
			if !called {
				t.Errorf("Expected next() to be called for %s", method)
			}
		})
	}
}

func TestCSRFMiddleware_UnsafeMethods_MissingOrigin(t *testing.T) {
	cfg := &CSRFConfig{
		CheckOrigin:  true,
		AllowOrigins: []string{},
		SiteURL:      "https://example.com",
	}
	csrfMw := NewCSRFMiddleware(cfg)

	unsafeMethods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range unsafeMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()
			ctx := middleware.NewContext(w, req)

			called := false
			err := csrfMw.Middleware(ctx, func() error {
				called = true
				return nil
			})

			if err == nil {
				t.Errorf("Expected error for %s without origin", method)
			}
			if called {
				t.Errorf("Expected next() NOT to be called for %s without origin", method)
			}
			if w.Code != http.StatusForbidden {
				t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
			}
		})
	}
}

func TestCSRFMiddleware_ValidOrigin(t *testing.T) {
	cfg := &CSRFConfig{
		CheckOrigin:  true,
		AllowOrigins: []string{},
		SiteURL:      "https://example.com",
	}
	csrfMw := NewCSRFMiddleware(cfg)

	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	ctx := middleware.NewContext(w, req)

	called := false
	err := csrfMw.Middleware(ctx, func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error for valid origin, got %v", err)
	}
	if !called {
		t.Error("Expected next() to be called for valid origin")
	}
}

func TestCSRFMiddleware_InvalidOrigin(t *testing.T) {
	cfg := &CSRFConfig{
		CheckOrigin:  true,
		AllowOrigins: []string{},
		SiteURL:      "https://example.com",
	}
	csrfMw := NewCSRFMiddleware(cfg)

	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	ctx := middleware.NewContext(w, req)

	called := false
	err := csrfMw.Middleware(ctx, func() error {
		called = true
		return nil
	})

	if err == nil {
		t.Error("Expected error for invalid origin")
	}
	if called {
		t.Error("Expected next() NOT to be called for invalid origin")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestCSRFMiddleware_AdditionalOrigins(t *testing.T) {
	cfg := &CSRFConfig{
		CheckOrigin:  true,
		AllowOrigins: []string{"https://api.example.com", "https://admin.example.com"},
		SiteURL:      "https://example.com",
	}
	csrfMw := NewCSRFMiddleware(cfg)

	origins := []string{
		"https://example.com",
		"https://api.example.com",
		"https://admin.example.com",
	}

	for _, origin := range origins {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("Origin", origin)
			w := httptest.NewRecorder()
			ctx := middleware.NewContext(w, req)

			called := false
			err := csrfMw.Middleware(ctx, func() error {
				called = true
				return nil
			})

			if err != nil {
				t.Errorf("Expected no error for %s, got %v", origin, err)
			}
			if !called {
				t.Errorf("Expected next() to be called for %s", origin)
			}
		})
	}
}

func TestCSRFMiddleware_LocalhostAllowed(t *testing.T) {
	cfg := &CSRFConfig{
		CheckOrigin:  true,
		AllowOrigins: []string{},
		SiteURL:      "https://example.com",
	}
	csrfMw := NewCSRFMiddleware(cfg)

	localhosts := []string{
		"http://localhost:3000",
		"http://localhost",
		"http://127.0.0.1:8080",
		"http://127.0.0.1",
	}

	for _, origin := range localhosts {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("Origin", origin)
			w := httptest.NewRecorder()
			ctx := middleware.NewContext(w, req)

			called := false
			err := csrfMw.Middleware(ctx, func() error {
				called = true
				return nil
			})

			if err != nil {
				t.Errorf("Expected no error for localhost %s, got %v", origin, err)
			}
			if !called {
				t.Errorf("Expected next() to be called for localhost %s", origin)
			}
		})
	}
}

func TestCSRFMiddleware_RefererFallback(t *testing.T) {
	cfg := &CSRFConfig{
		CheckOrigin:  true,
		AllowOrigins: []string{},
		SiteURL:      "https://example.com",
	}
	csrfMw := NewCSRFMiddleware(cfg)

	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Referer", "https://example.com/page")
	w := httptest.NewRecorder()
	ctx := middleware.NewContext(w, req)

	called := false
	err := csrfMw.Middleware(ctx, func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error with valid referer, got %v", err)
	}
	if !called {
		t.Error("Expected next() to be called with valid referer")
	}
}

func TestCSRFMiddleware_Disabled(t *testing.T) {
	cfg := &CSRFConfig{
		CheckOrigin:  false,
		AllowOrigins: []string{},
		SiteURL:      "https://example.com",
	}
	csrfMw := NewCSRFMiddleware(cfg)

	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()
	ctx := middleware.NewContext(w, req)

	called := false
	err := csrfMw.Middleware(ctx, func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error when CSRF disabled, got %v", err)
	}
	if !called {
		t.Error("Expected next() to be called when CSRF disabled")
	}
}

func TestCSRFMiddleware_CaseInsensitive(t *testing.T) {
	cfg := &CSRFConfig{
		CheckOrigin:  true,
		AllowOrigins: []string{},
		SiteURL:      "https://EXAMPLE.com",
	}
	csrfMw := NewCSRFMiddleware(cfg)

	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Origin", "https://example.COM")
	w := httptest.NewRecorder()
	ctx := middleware.NewContext(w, req)

	called := false
	err := csrfMw.Middleware(ctx, func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error for case-insensitive match, got %v", err)
	}
	if !called {
		t.Error("Expected next() to be called for case-insensitive match")
	}
}

func TestCSRFMiddleware_TrailingSlash(t *testing.T) {
	cfg := &CSRFConfig{
		CheckOrigin:  true,
		AllowOrigins: []string{},
		SiteURL:      "https://example.com/",
	}
	csrfMw := NewCSRFMiddleware(cfg)

	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	ctx := middleware.NewContext(w, req)

	called := false
	err := csrfMw.Middleware(ctx, func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error with trailing slash normalization, got %v", err)
	}
	if !called {
		t.Error("Expected next() to be called with trailing slash normalization")
	}
}

func TestCSRFMiddleware_ContentTypeFiltering(t *testing.T) {
	cfg := &CSRFConfig{
		CheckOrigin:  true,
		AllowOrigins: []string{},
		SiteURL:      "https://example.com",
	}
	csrfMw := NewCSRFMiddleware(cfg)

	tests := []struct {
		name        string
		contentType string
		shouldCheck bool
	}{
		{"application/json", "application/json", false},
		{"application/x-www-form-urlencoded", "application/x-www-form-urlencoded", true},
		{"multipart/form-data", "multipart/form-data", true},
		{"text/plain", "text/plain", true},
		{"empty", "", true},
		{"with charset", "application/x-www-form-urlencoded; charset=utf-8", true},
		{"json with charset", "application/json; charset=utf-8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()
			ctx := middleware.NewContext(w, req)

			called := false
			err := csrfMw.Middleware(ctx, func() error {
				called = true
				return nil
			})

			if tt.shouldCheck {
				if err == nil {
					t.Errorf("Expected error (no origin), got nil")
				}
				if called {
					t.Errorf("Expected next() NOT to be called")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if !called {
					t.Errorf("Expected next() to be called")
				}
			}
		})
	}
}
