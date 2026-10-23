---
topic: email
last_verified: 2026-10-23
sources:
  - internal/usecase/email.go
  - internal/infrastructure/email/mailjet.go
  - internal/infrastructure/email/welcome.go
  - internal/infrastructure/email/preview.go
  - internal/infrastructure/email/templates/welcome.html
  - internal/infrastructure/email/templates/welcome.txt
  - internal/infrastructure/queue/handlers.go
  - internal/infrastructure/queue/tasks.go
  - internal/bootstrap/bootstrap.go
  - internal/server/server.go
  - internal/transport/handlers/handler.go
  - internal/transport/handlers/routes.go
---

# Email (Mailjet)

## Overview

`usecase.EmailSender` is the interface all email sending goes through:

```go
// internal/usecase/email.go
type EmailSender interface {
    SendWelcomeEmail(ctx context.Context, toEmail, toName string) error
}
```

The interface lives in `usecase/` because that is the layer that depends on it — handlers and future use-case logic call `EmailSender` without knowing about Mailjet.

`App.EmailSender` is `nil` when `MAILJET_API_KEY` is omitted, so callers must nil-guard before use.

## Implementation

`MailjetSender` in `internal/infrastructure/email/mailjet.go` implements the interface using the Mailjet Send API v3.1.

### Constructors

```go
// Production — sends real email
func NewMailjetSender(apiKey, secretKey, fromEmail, fromName string, baseURL ...string) *MailjetSender

// Sandbox — Mailjet validates the payload but does not deliver; for integration tests
func NewSandboxSender(apiKey, secretKey, fromEmail, fromName string) *MailjetSender
```

The optional `baseURL` variadic on `NewMailjetSender` overrides the Mailjet API endpoint, which allows unit tests to point the sender at an `httptest.Server`.

### Templates

Each email has two templates in `internal/infrastructure/email/templates/`, embedded at compile time (`//go:embed templates/*` in `welcome.go`) and parsed once at startup:

- `welcome.html` — Go `html/template`, so values people typed (names) are escaped. Table layout with inline styles, as email clients need.
- `welcome.txt` — Go `text/template`, the plain-text part (sent alongside the HTML; some clients and people read only that).

`renderWelcome(name)` (`welcome.go`) fills `welcomeData` (`Name`, "there" when empty) and returns a `message{Subject, HTML, Text}`. The sender sends that message, and the preview below shows it, so what you preview is exactly what is sent.

### Previewing the templates

With the backend running locally (`make watch`), open **http://localhost:8080/dev/emails**. It shows every email rendered with sample data:

- A sidebar of samples (`previewSamples` in `preview.go`), one per way a template can come out (e.g. with and without a name). Add a sample when a template gains a branch.
- **HTML / Text** switches between the two parts; **Desktop / Phone** frames the HTML at full width or 375px.
- `/dev/emails/<sample>.html` and `.txt` serve each part on its own.

Save a template and Air rebuilds the backend (it watches `.html` and `.txt`); the open page polls `/dev/emails/build`, which changes on every restart, and reloads itself. No separate server to run, and no Mailjet keys needed.

It is only registered in debug mode (`ENV` other than `staging`/`production`), like `/admin/queues`: `server.NewServer` passes `email.NewPreview("/dev/emails")` to `Handler.WithEmailPreview`.

### How welcome email is triggered

The welcome email is sent via the Asynq queue (not directly from a handler). The Asynq
task handler `NewHandleWelcomeEmail(sender)` in `internal/infrastructure/queue/handlers.go`
calls `sender.SendWelcomeEmail` with the name from `WelcomeEmailPayload.Name`.
When `WelcomeEmailPayload.Name` is empty the handler falls back to `WelcomeEmailPayload.Email`
as the display name.

```go
// Triggered when a TypeWelcomeEmail task is dequeued.
func NewHandleWelcomeEmail(sender usecase.EmailSender) asynq.HandlerFunc
```

`sender` is nil when `MAILJET_API_KEY` is not set; the handler no-ops gracefully in that case.

### Wiring (bootstrap)

`bootstrap.Run` constructs the sender when both `MAILJET_API_KEY` and `MAILJET_SECRET_KEY` are non-empty:

```go
var emailSender usecase.EmailSender
if cfg.MailjetAPIKey != "" && cfg.MailjetSecretKey != "" {
    emailSender = email.NewMailjetSender(
        cfg.MailjetAPIKey,
        cfg.MailjetSecretKey,
        cfg.FromEmail,
        cfg.FromName,
    )
}
```

`emailSender` is stored on `bootstrap.App.EmailSender`. `server.NewServer` passes it to `handlers.NewHandler` as the last argument.

## Env vars

| Variable | Required | Description |
|---|---|---|
| `MAILJET_API_KEY` | No | Mailjet API key. Omit (or leave empty) to disable all email sending. |
| `MAILJET_SECRET_KEY` | No | Mailjet secret key. Must be set alongside `MAILJET_API_KEY`. |
| `FROM_EMAIL` | No | Verified sender address — e.g. `no-reply@example.com`. |
| `FROM_NAME` | No | Sender display name — e.g. `MyApp`. |

When `MAILJET_API_KEY` or `MAILJET_SECRET_KEY` is empty, `App.EmailSender` is `nil` and no email is sent. The other two vars are only read when the sender is initialised.

## How to add a new email type

1. **Add a method to the interface** in `internal/usecase/email.go`:
   ```go
   SendPasswordResetEmail(ctx context.Context, toEmail, resetURL string) error
   ```

2. **Add the templates** `internal/infrastructure/email/templates/password_reset.html` and `password_reset.txt`, and a render function like `renderWelcome` that returns a `message`. Use Go template syntax (`{{.FieldName}}`) for dynamic values.

3. **Implement the method** on `MailjetSender` in `internal/infrastructure/email/mailjet.go`. Follow the `SendWelcomeEmail` pattern: render the message, populate `mailjet.InfoMessagesV31` (`Subject`, `TextPart`, `HTMLPart`), call `client.SendMailV31`.

4. **Add it to the preview**: give `previewSample` a way to render the new email (e.g. a render func per sample) and add samples to `previewSamples` in `preview.go`, so it shows at `/dev/emails`.

5. **Update the test double** in `internal/usecase/email_usecase_test.go` to add the new method stub so `mockEmailSender` continues to satisfy the interface at compile time.

6. **Write tests** — see the Testing section below.

## Testing

### Unit test — interface contract (`internal/usecase/`)

Test that a mock satisfies the interface and propagates errors correctly. No network calls.

```go
type mockEmailSender struct {
    calledWithEmail string
    err             error
}

func (m *mockEmailSender) SendWelcomeEmail(_ context.Context, toEmail, toName string) error {
    m.calledWithEmail = toEmail
    return m.err
}

var _ usecase.EmailSender = (*mockEmailSender)(nil) // compile-time check
```

See `internal/usecase/email_usecase_test.go` for the full example.

### Unit test — HTTP error propagation (`internal/infrastructure/email/`)

Use `httptest.NewServer` and pass its URL as the `baseURL` override to `NewMailjetSender`. No real credentials needed.

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
    w.WriteHeader(http.StatusUnauthorized)
    json.NewEncoder(w).Encode(map[string]any{"StatusCode": 401})
}))
defer srv.Close()

sender := email.NewMailjetSender("bad-key", "bad-secret", "no-reply@example.com", "Test", srv.URL+"/v3")
err := sender.SendWelcomeEmail(context.Background(), "user@example.com", "User")
// assert err != nil
```

### Unit tests — rendering and preview (`internal/infrastructure/email/`)

- `welcome_internal_test.go`: `renderWelcome` — subject, both parts greet by name, "there" without one, names escaped in HTML.
- `preview_test.go`: the preview lists every sample, serves each part exactly as `renderWelcome` renders it, 404s unknown samples, and keeps one build token per process.
- `mailjet_test.go` (`SendsBothParts`): the request carries the rendered subject, `HTMLPart` and `TextPart`.
- `internal/transport/handlers/email_preview_handler_test.go`: `/dev/emails` is served in debug mode only.

### Integration test — live Mailjet sandbox

Uses `NewSandboxSender` which sets `SandBoxMode: true` on the Mailjet request. Mailjet validates the payload and returns a success response without delivering the email.

```go
func TestMailjetSender_SendWelcomeEmail_Sandbox(t *testing.T) {
    apiKey := os.Getenv("MAILJET_API_KEY")
    secretKey := os.Getenv("MAILJET_SECRET_KEY")
    if apiKey == "" || secretKey == "" {
        t.Skip("MAILJET_API_KEY and MAILJET_SECRET_KEY not set — skipping Mailjet sandbox integration test")
    }

    sender := email.NewSandboxSender(apiKey, secretKey, "no-reply@example.com", "MyApp Test")
    err := sender.SendWelcomeEmail(context.Background(), "sandbox@mailjet.com", "Sandbox User")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

The skip guard means this test is silent in CI unless the credentials are injected as secrets.
