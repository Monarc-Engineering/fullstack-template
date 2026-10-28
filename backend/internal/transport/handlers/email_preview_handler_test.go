package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// previewStub records the path it was asked for.
type previewStub struct{ path string }

func (p *previewStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.path = r.URL.Path
	w.WriteHeader(http.StatusOK)
}

func serveEmailPreview(t *testing.T, mode string, preview http.Handler, path string) int {
	t.Helper()
	prev := gin.Mode()
	gin.SetMode(mode)
	t.Cleanup(func() { gin.SetMode(prev) })

	h := (&Handler{}).WithEmailPreview(preview)
	w := httptest.NewRecorder()
	h.RegisterRoutes(0, 0, "", []string{"http://localhost:3000"}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Code
}

func TestEmailPreview_ServedInDebugMode(t *testing.T) {
	for _, path := range []string{"/dev/emails", "/dev/emails/challenge.html"} {
		stub := &previewStub{}
		if code := serveEmailPreview(t, gin.DebugMode, stub, path); code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", path, code)
		}
		if stub.path != path {
			t.Errorf("%s: preview got path %q", path, stub.path)
		}
	}
}

func TestEmailPreview_NotServedInRelease(t *testing.T) {
	if code := serveEmailPreview(t, gin.ReleaseMode, &previewStub{}, "/dev/emails"); code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}

func TestEmailPreview_NotServedWithoutPreview(t *testing.T) {
	if code := serveEmailPreview(t, gin.DebugMode, nil, "/dev/emails"); code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}
