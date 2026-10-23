package email

import (
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func getPreview(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestPreview_IndexListsEverySample(t *testing.T) {
	h := NewPreview("/dev/emails")
	w := getPreview(t, h, "/dev/emails")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	body := w.Body.String()
	for _, s := range previewSamples {
		msg, err := renderWelcome(s.Name)
		if err != nil {
			t.Fatalf("render %s: %v", s.ID, err)
		}
		if !strings.Contains(body, `href="/dev/emails?email=`+s.ID) {
			t.Errorf("index has no link to %s", s.ID)
		}
		if !strings.Contains(body, html.EscapeString(msg.Subject)) {
			t.Errorf("index doesn't show %s's subject %q", s.ID, msg.Subject)
		}
	}
}

func TestPreview_ShowsTheChosenSample(t *testing.T) {
	h := NewPreview("/dev/emails")
	s := previewSamples[len(previewSamples)-1]

	body := getPreview(t, h, "/dev/emails?email="+s.ID).Body.String()
	if !strings.Contains(body, `src="/dev/emails/`+s.ID+`.html"`) {
		t.Errorf("page doesn't frame %s's HTML", s.ID)
	}
}

func TestPreview_TextView(t *testing.T) {
	h := NewPreview("/dev/emails")
	s := previewSamples[0]
	msg, _ := renderWelcome(s.Name)

	body := getPreview(t, h, "/dev/emails?email="+s.ID+"&view=text").Body.String()
	firstLine := strings.SplitN(msg.Text, "\n", 2)[0]
	if !strings.Contains(body, html.EscapeString(firstLine)) {
		t.Errorf("text view doesn't show the plain-text part (%q)", firstLine)
	}
}

func TestPreview_ServesTheRenderedParts(t *testing.T) {
	h := NewPreview("/dev/emails")
	for _, s := range previewSamples {
		msg, err := renderWelcome(s.Name)
		if err != nil {
			t.Fatalf("render %s: %v", s.ID, err)
		}

		w := getPreview(t, h, "/dev/emails/"+s.ID+".html")
		if w.Code != http.StatusOK || w.Body.String() != msg.HTML {
			t.Errorf("%s.html: status %d, body is not what Mailjet would get", s.ID, w.Code)
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("%s.html: Content-Type = %q", s.ID, ct)
		}

		w = getPreview(t, h, "/dev/emails/"+s.ID+".txt")
		if w.Code != http.StatusOK || w.Body.String() != msg.Text {
			t.Errorf("%s.txt: status %d, body is not what Mailjet would get", s.ID, w.Code)
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
			t.Errorf("%s.txt: Content-Type = %q", s.ID, ct)
		}
	}
}

func TestPreview_UnknownSampleIsNotFound(t *testing.T) {
	h := NewPreview("/dev/emails")
	for _, path := range []string{"/dev/emails?email=nope", "/dev/emails/nope.html", "/dev/emails/nope.txt", "/dev/emails/other"} {
		if w := getPreview(t, h, path); w.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", path, w.Code)
		}
	}
}

func TestPreview_BuildIsStableForOneProcess(t *testing.T) {
	h := NewPreview("/dev/emails")
	first := getPreview(t, h, "/dev/emails/build").Body.String()
	if first == "" {
		t.Fatal("build is empty")
	}
	if again := getPreview(t, h, "/dev/emails/build").Body.String(); again != first {
		t.Errorf("build changed from %q to %q without a restart", first, again)
	}
}

func TestPreviewSamples_HaveUniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range previewSamples {
		if s.ID == "" || seen[s.ID] {
			t.Errorf("sample ID %q is empty or repeated", s.ID)
		}
		seen[s.ID] = true
	}
}
