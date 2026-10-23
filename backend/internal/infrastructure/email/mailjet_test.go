package email_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"backend/internal/infrastructure/email"
)

// TestMain is the entry point for this test package.
// No external containers are needed — real Mailjet sandbox API is used when
// credentials are available; otherwise the sandbox test is skipped.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// TestMailjetSender_SendWelcomeEmail_Sandbox calls the real Mailjet Send API
// in sandbox mode (email is validated but not delivered). It is skipped when
// MAILJET_API_KEY / MAILJET_SECRET_KEY are not set in the environment.
func TestMailjetSender_SendWelcomeEmail_Sandbox(t *testing.T) {
	apiKey := os.Getenv("MAILJET_API_KEY")
	secretKey := os.Getenv("MAILJET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		t.Skip("MAILJET_API_KEY and MAILJET_SECRET_KEY not set — skipping Mailjet sandbox integration test")
	}

	fromEmail := os.Getenv("FROM_EMAIL")
	fromName := os.Getenv("FROM_NAME")
	if fromEmail == "" {
		fromEmail = "no-reply@example.com"
	}
	if fromName == "" {
		fromName = "MyApp Test"
	}

	// withSandbox is unexported but we can exercise it by keeping the sender
	// internal to this package test; instead we call the exported constructor
	// and rely on a helper that forces sandbox mode (see newSandboxSender).
	sender := email.NewSandboxSender(apiKey, secretKey, fromEmail, fromName)

	err := sender.SendWelcomeEmail(context.Background(), "sandbox@mailjet.com", "Sandbox User")
	if err != nil {
		t.Fatalf("SendWelcomeEmail sandbox: unexpected error: %v", err)
	}
}

// TestMailjetSender_SendWelcomeEmail_Non200_ReturnsError uses an httptest
// server that returns 401 Unauthorized to verify the sender propagates errors.
func TestMailjetSender_SendWelcomeEmail_Non200_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"ErrorInfo":    "api key invalid",
			"ErrorMessage": "Unauthorized",
			"StatusCode":   401,
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	// Pass the test server URL as the baseURL override.
	sender := email.NewMailjetSender("bad-key", "bad-secret", "no-reply@example.com", "Test", srv.URL+"/v3")

	err := sender.SendWelcomeEmail(context.Background(), "user@example.com", "User")
	if err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
}

// TestMailjetSender_SendWelcomeEmail_Success uses an httptest server that
// returns a valid Mailjet v3.1 success response body.
func TestMailjetSender_SendWelcomeEmail_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Minimal valid Mailjet v3.1 response.
		if err := json.NewEncoder(w).Encode(map[string]any{
			"Messages": []map[string]any{
				{
					"Status": "success",
					"To": []map[string]any{
						{
							"Email":       "user@example.com",
							"MessageUUID": "abc-123",
							"MessageID":   1111111111111111,
							"MessageHref": "https://api.mailjet.com/v3/REST/message/1111111111111111",
						},
					},
				},
			},
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	sender := email.NewMailjetSender("key", "secret", "no-reply@example.com", "Test", srv.URL+"/v3")

	err := sender.SendWelcomeEmail(context.Background(), "user@example.com", "User")
	if err != nil {
		t.Fatalf("SendWelcomeEmail: unexpected error: %v", err)
	}
}

// TestMailjetSender_SendWelcomeEmail_SendsBothParts checks the request carries
// the rendered subject, HTML and plain-text parts.
func TestMailjetSender_SendWelcomeEmail_SendsBothParts(t *testing.T) {
	var got struct {
		Messages []struct {
			Subject  string
			TextPart string
			HTMLPart string
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"Messages": []map[string]any{{"Status": "success"}}})
	}))
	defer srv.Close()

	sender := email.NewMailjetSender("key", "secret", "no-reply@example.com", "Test", srv.URL+"/v3")
	if err := sender.SendWelcomeEmail(context.Background(), "user@example.com", "Ada"); err != nil {
		t.Fatalf("SendWelcomeEmail: %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
	m := got.Messages[0]
	if m.Subject != "Welcome to MyApp!" {
		t.Errorf("Subject = %q", m.Subject)
	}
	if !strings.Contains(m.HTMLPart, "Welcome, Ada!") || !strings.Contains(m.TextPart, "Welcome, Ada!") {
		t.Errorf("expected both parts to greet Ada; HTML %q, text %q", m.HTMLPart, m.TextPart)
	}
}
