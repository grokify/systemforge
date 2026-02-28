package bff

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewOriginValidator(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
		AllowedHosts:   []string{"localhost:3000"},
		SkipMethods:    []string{"GET", "HEAD"},
	})

	if !v.allowedOrigins["https://example.com"] {
		t.Error("expected https://example.com to be allowed")
	}
	if !v.allowedOrigins["https://app.example.com"] {
		t.Error("expected https://app.example.com to be allowed")
	}
	if !v.allowedHosts["localhost:3000"] {
		t.Error("expected localhost:3000 to be allowed")
	}
	if !v.skipMethods["GET"] {
		t.Error("expected GET to be skipped")
	}
	if !v.skipMethods["HEAD"] {
		t.Error("expected HEAD to be skipped")
	}
}

func TestOriginValidator_ValidateRequest_AllowedOrigin(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins: []string{"https://example.com"},
	})

	req := httptest.NewRequest("POST", "/api/resource", nil)
	req.Header.Set("Origin", "https://example.com")

	if !v.ValidateRequest(req) {
		t.Error("ValidateRequest() = false, want true for allowed origin")
	}
}

func TestOriginValidator_ValidateRequest_DisallowedOrigin(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins: []string{"https://example.com"},
	})

	req := httptest.NewRequest("POST", "/api/resource", nil)
	req.Header.Set("Origin", "https://evil.com")

	if v.ValidateRequest(req) {
		t.Error("ValidateRequest() = true, want false for disallowed origin")
	}
}

func TestOriginValidator_ValidateRequest_CaseInsensitive(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins: []string{"https://Example.COM"},
	})

	req := httptest.NewRequest("POST", "/api/resource", nil)
	req.Header.Set("Origin", "https://example.com")

	if !v.ValidateRequest(req) {
		t.Error("ValidateRequest() should be case-insensitive")
	}
}

func TestOriginValidator_ValidateRequest_AllowedHost(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedHosts: []string{"localhost:3000"},
	})

	req := httptest.NewRequest("POST", "/api/resource", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	if !v.ValidateRequest(req) {
		t.Error("ValidateRequest() = false, want true for allowed host")
	}
}

func TestOriginValidator_ValidateRequest_Referer(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins: []string{"https://example.com"},
		CheckReferer:   true,
	})

	req := httptest.NewRequest("POST", "/api/resource", nil)
	// No Origin header, but has Referer
	req.Header.Set("Referer", "https://example.com/page")

	if !v.ValidateRequest(req) {
		t.Error("ValidateRequest() = false, want true for allowed referer")
	}
}

func TestOriginValidator_ValidateRequest_DisallowedReferer(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins: []string{"https://example.com"},
		CheckReferer:   true,
	})

	req := httptest.NewRequest("POST", "/api/resource", nil)
	req.Header.Set("Referer", "https://evil.com/page")

	if v.ValidateRequest(req) {
		t.Error("ValidateRequest() = true, want false for disallowed referer")
	}
}

func TestOriginValidator_ValidateRequest_NoOriginAllowed(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins:     []string{"https://example.com"},
		AllowMissingOrigin: true,
	})

	req := httptest.NewRequest("POST", "/api/resource", nil)
	// No Origin or Referer header

	if !v.ValidateRequest(req) {
		t.Error("ValidateRequest() = false, want true when AllowMissingOrigin is true")
	}
}

func TestOriginValidator_ValidateRequest_NoOriginDenied(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins:     []string{"https://example.com"},
		AllowMissingOrigin: false,
	})

	req := httptest.NewRequest("POST", "/api/resource", nil)
	// No Origin or Referer header

	if v.ValidateRequest(req) {
		t.Error("ValidateRequest() = true, want false when AllowMissingOrigin is false")
	}
}

func TestOriginValidator_Middleware_Allowed(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins: []string{"https://example.com"},
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := v.Middleware()

	req := httptest.NewRequest("POST", "/api/resource", nil)
	req.Header.Set("Origin", "https://example.com")

	rr := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestOriginValidator_Middleware_Denied(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins: []string{"https://example.com"},
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for denied origin")
	})

	middleware := v.Middleware()

	req := httptest.NewRequest("POST", "/api/resource", nil)
	req.Header.Set("Origin", "https://evil.com")

	rr := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestOriginValidator_Middleware_SkipMethods(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins: []string{"https://example.com"},
		SkipMethods:    []string{"GET", "HEAD"},
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := v.Middleware()

	// GET without Origin should pass
	req := httptest.NewRequest("GET", "/api/resource", nil)
	rr := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GET Status = %d, want %d", rr.Code, http.StatusOK)
	}

	// HEAD without Origin should pass
	req = httptest.NewRequest("HEAD", "/api/resource", nil)
	rr = httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("HEAD Status = %d, want %d", rr.Code, http.StatusOK)
	}

	// POST without Origin should fail
	req = httptest.NewRequest("POST", "/api/resource", nil)
	rr = httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("POST Status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestOriginValidator_Middleware_CustomError(t *testing.T) {
	customErrorCalled := false
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins: []string{"https://example.com"},
		OnError: func(w http.ResponseWriter, r *http.Request) {
			customErrorCalled = true
			http.Error(w, "Custom error", http.StatusTeapot)
		},
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	middleware := v.Middleware()

	req := httptest.NewRequest("POST", "/api/resource", nil)
	req.Header.Set("Origin", "https://evil.com")

	rr := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	if !customErrorCalled {
		t.Error("Custom error handler should be called")
	}
	if rr.Code != http.StatusTeapot {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusTeapot)
	}
}

func TestOriginMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := OriginMiddleware("https://example.com", "https://app.example.com")

	// Allowed origin
	req := httptest.NewRequest("POST", "/api", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Disallowed origin
	req = httptest.NewRequest("POST", "/api", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr = httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestDefaultOriginConfig(t *testing.T) {
	cfg := DefaultOriginConfig()

	if !cfg.CheckReferer {
		t.Error("CheckReferer = false, want true")
	}
	if cfg.AllowMissingOrigin {
		t.Error("AllowMissingOrigin = true, want false")
	}
}

func TestOriginValidator_InvalidReferer(t *testing.T) {
	v := NewOriginValidator(OriginConfig{
		AllowedOrigins: []string{"https://example.com"},
		CheckReferer:   true,
	})

	req := httptest.NewRequest("POST", "/api/resource", nil)
	req.Header.Set("Referer", "not-a-valid-url")

	if v.ValidateRequest(req) {
		t.Error("ValidateRequest() = true, want false for invalid referer")
	}
}
