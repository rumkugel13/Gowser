package url

import (
	"testing"
)

func TestScheme(t *testing.T) {
	u, _ := NewURL("http://example.com/path")
	if u.scheme != "http" {
		t.Errorf("Expected scheme 'http', got '%s'", u.scheme)
	}
}

func TestHTTPSScheme(t *testing.T) {
	u, _ := NewURL("https://example.com/path")
	if u.scheme != "https" {
		t.Errorf("Expected scheme 'https', got '%s'", u.scheme)
	}
}

func TestDefaultPort(t *testing.T) {
	u, _ := NewURL("http://example.com/path")
	if u.port != 80 {
		t.Errorf("Expected default port 80 for HTTP, got %d", u.port)
	}

	u, _ = NewURL("https://example.com/path")
	if u.port != 443 {
		t.Errorf("Expected default port 443 for HTTPS, got %d", u.port)
	}
}

func TestCustomPort(t *testing.T) {
	u, _ := NewURL("http://example.com:8080/path")
	if u.port != 8080 {
		t.Errorf("Expected port 8080, got %d", u.port)
	}

	u, _ = NewURL("https://example.com:8443/path")
	if u.port != 8443 {
		t.Errorf("Expected port 8443, got %d", u.port)
	}
}

func TestUnsupportedPort(t *testing.T) {
	_, err := NewURL("http://example.com:invalid/path")
	if err == nil {
		t.Error("Expected error for unsupported port, but did not error")
	}
}

func TestUnsupportedScheme(t *testing.T) {
	_, err := NewURL("ftp://example.com/path")
	if err == nil {
		t.Error("Expected error for unsupported scheme, but did not error")
	}
}

func TestHost(t *testing.T) {
	u, _ := NewURL("http://example.com/path")
	if u.host != "example.com" {
		t.Errorf("Expected host 'example.com', got '%s'", u.host)
	}
}

func TestPath(t *testing.T) {
	u, _ := NewURL("http://example.com/path/to/resource")
	if u.path != "/path/to/resource" {
		t.Errorf("Expected path '/path/to/resource', got '%s'", u.path)
	}
}

func TestMissingPath(t *testing.T) {
	u, _ := NewURL("http://example.com")
	if u.path != "/" {
		t.Errorf("Expected path '/', got '%s'", u.path)
	}
}

func TestInvalidURL(t *testing.T) {
	_, err := NewURL("invalid-url")
	if err == nil {
		t.Error("Expected error for invalid URL, but did not error")
	}
}

func TestEmptyURL(t *testing.T) {
	_, err := NewURL("")
	if err == nil {
		t.Error("Expected error for empty URL, but did not error")
	}
}

func TestURLWithTrailingSlash(t *testing.T) {
	u, _ := NewURL("http://example.com/path/")
	if u.path != "/path/" {
		t.Errorf("Expected path '/path/', got '%s'", u.path)
	}
}

func TestRequest(t *testing.T) {
	u, _ := NewURL("http://example.com/path")
	_, response, _ := u.Request(u, "")
	if string(response) == "" {
		t.Error("Expected non-empty response from Request()")
	}
}
