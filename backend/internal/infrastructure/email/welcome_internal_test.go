package email

import (
	"strings"
	"testing"
)

func TestRenderWelcome(t *testing.T) {
	msg, err := renderWelcome("Ada")
	if err != nil {
		t.Fatalf("renderWelcome: %v", err)
	}
	if msg.Subject != "Welcome to MyApp!" {
		t.Errorf("Subject = %q", msg.Subject)
	}
	if !strings.Contains(msg.HTML, "Welcome, Ada!") {
		t.Error("HTML part doesn't greet them by name")
	}
	if !strings.Contains(msg.Text, "Welcome, Ada!") {
		t.Error("plain-text part doesn't greet them by name")
	}
}

func TestRenderWelcome_WithoutAName(t *testing.T) {
	msg, err := renderWelcome("")
	if err != nil {
		t.Fatalf("renderWelcome: %v", err)
	}
	if !strings.Contains(msg.HTML, "Welcome, there!") || !strings.Contains(msg.Text, "Welcome, there!") {
		t.Error(`expected "there" in place of a missing name`)
	}
}

func TestRenderWelcome_EscapesTheNameInHTML(t *testing.T) {
	msg, err := renderWelcome("<b>Ada</b>")
	if err != nil {
		t.Fatalf("renderWelcome: %v", err)
	}
	if strings.Contains(msg.HTML, "<b>Ada</b>") {
		t.Error("the name was not escaped in the HTML part")
	}
}
