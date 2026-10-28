package email

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	texttemplate "text/template"
)

//go:embed templates/*
var templateFS embed.FS

var (
	welcomeHTML = htmltemplate.Must(htmltemplate.ParseFS(templateFS, "templates/welcome.html"))
	welcomeText = texttemplate.Must(texttemplate.ParseFS(templateFS, "templates/welcome.txt"))
)

// message is a rendered email: what Mailjet sends, and what the preview shows.
type message struct {
	Subject string
	HTML    string
	Text    string
}

// welcomeData is what the welcome templates show.
type welcomeData struct {
	Name string
}

// renderWelcome renders the welcome email for name ("there" when empty).
func renderWelcome(name string) (message, error) {
	data := welcomeData{Name: name}
	if data.Name == "" {
		data.Name = "there"
	}

	var html, text bytes.Buffer
	if err := welcomeHTML.Execute(&html, data); err != nil {
		return message{}, fmt.Errorf("html: %w", err)
	}
	if err := welcomeText.Execute(&text, data); err != nil {
		return message{}, fmt.Errorf("text: %w", err)
	}
	return message{Subject: "Welcome to MyApp!", HTML: html.String(), Text: text.String()}, nil
}
