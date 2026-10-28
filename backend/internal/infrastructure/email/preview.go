package email

import (
	htmltemplate "html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// previewSample is one email the preview shows, with a made-up recipient.
type previewSample struct {
	ID    string
	Label string
	Name  string
}

// previewSamples covers each way the templates can come out. Add one when a
// template gains a branch, and samples for each new email.
var previewSamples = []previewSample{
	{"welcome", "Welcome", "Ada"},
	{"welcome-no-name", "Welcome, no name", ""},
}

func findPreviewSample(id string) (previewSample, bool) {
	for _, s := range previewSamples {
		if s.ID == id {
			return s, true
		}
	}
	return previewSample{}, false
}

// preview serves the email templates rendered with previewSamples.
type preview struct {
	base string
	// build changes each time the process starts, so an open page knows to
	// reload once Air has rebuilt with an edited template.
	build string
}

// NewPreview returns a page, mounted at base (e.g. "/dev/emails"), showing
// each email as Mailjet would get it, rendered by the same code that sends it:
//
//	base                  the page (?email=<id>, &view=text, &width=phone)
//	base/<id>.html        the HTML part
//	base/<id>.txt         the plain-text part
//	base/build            changes on restart; the page polls it to reload
//
// It is for local development only: it is never registered in release mode.
func NewPreview(base string) http.Handler {
	return &preview{
		base:  strings.TrimRight(base, "/"),
		build: strconv.FormatInt(time.Now().UnixNano(), 36),
	}
}

func (p *preview) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, p.base)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rest = strings.TrimPrefix(rest, "/")
	w.Header().Set("Cache-Control", "no-store")

	switch {
	case rest == "":
		p.page(w, r)
	case rest == "build":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(p.build))
	case strings.HasSuffix(rest, ".html"):
		p.part(w, r, strings.TrimSuffix(rest, ".html"), "text/html; charset=utf-8", func(m message) string { return m.HTML })
	case strings.HasSuffix(rest, ".txt"):
		p.part(w, r, strings.TrimSuffix(rest, ".txt"), "text/plain; charset=utf-8", func(m message) string { return m.Text })
	default:
		http.NotFound(w, r)
	}
}

// part serves one rendered part of sample id.
func (p *preview) part(w http.ResponseWriter, r *http.Request, id, contentType string, pick func(message) string) {
	s, ok := findPreviewSample(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	msg, err := renderWelcome(s.Name)
	if err != nil {
		http.Error(w, "render "+id+": "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write([]byte(pick(msg)))
}

type previewNavItem struct {
	ID, Label, Subject string
	Active             bool
}

type previewPage struct {
	Base, Build string
	Nav         []previewNavItem
	ID, To      string
	Subject     string
	Text        string
	TextView    bool
	Phone       bool
	Err         string
}

// previewRecipient is who a sample is addressed to, as the page shows it.
func previewRecipient(name string) string {
	if name == "" {
		return "(no name)"
	}
	return name
}

func (p *preview) page(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("email")
	if id == "" {
		id = previewSamples[0].ID
	}
	current, ok := findPreviewSample(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	data := previewPage{
		Base:     p.base,
		Build:    p.build,
		ID:       current.ID,
		To:       previewRecipient(current.Name),
		TextView: q.Get("view") == "text",
		Phone:    q.Get("width") == "phone",
	}
	for _, s := range previewSamples {
		item := previewNavItem{ID: s.ID, Label: s.Label, Active: s.ID == current.ID}
		msg, err := renderWelcome(s.Name)
		if err != nil {
			item.Subject = "Error: " + err.Error()
		} else {
			item.Subject = msg.Subject
		}
		if item.Active {
			data.Subject, data.Text = msg.Subject, msg.Text
			if err != nil {
				data.Err = err.Error()
			}
		}
		data.Nav = append(data.Nav, item)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := previewPageTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var previewPageTemplate = htmltemplate.Must(htmltemplate.New("preview").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Email preview</title>
<style>
  * { box-sizing: border-box; }
  body { margin: 0; font: 14px/1.4 system-ui, sans-serif; color: #1c1b1f; background: #f4f2f7; display: flex; min-height: 100vh; }
  nav { width: 280px; flex: none; background: #fff; border-right: 1px solid #e3dfe8; padding: 16px 0; }
  nav h1 { font-size: 13px; text-transform: uppercase; letter-spacing: .06em; color: #6b6474; margin: 0 16px 12px; }
  nav a { display: block; padding: 10px 16px; color: inherit; text-decoration: none; border-left: 3px solid transparent; }
  nav a:hover { background: #f4f2f7; }
  nav a.active { background: #efe8fb; border-left-color: #6a3fb5; }
  nav small { display: block; color: #6b6474; margin-top: 2px; }
  main { flex: 1; min-width: 0; padding: 20px 24px; display: flex; flex-direction: column; gap: 12px; }
  header { display: flex; flex-wrap: wrap; align-items: center; gap: 12px 24px; }
  header dl { margin: 0; display: grid; grid-template-columns: auto 1fr; gap: 2px 10px; }
  header dt { color: #6b6474; }
  header dd { margin: 0; font-weight: 600; }
  .toggles { margin-left: auto; display: flex; gap: 8px; }
  .toggle { display: inline-flex; border: 1px solid #d6d0de; border-radius: 6px; overflow: hidden; background: #fff; }
  .toggle a { padding: 6px 12px; color: inherit; text-decoration: none; }
  .toggle a.on { background: #6a3fb5; color: #fff; }
  .frame { flex: 1; display: flex; justify-content: center; }
  iframe { border: 1px solid #d6d0de; border-radius: 8px; background: #fff; width: 100%; min-height: 80vh; }
  iframe.phone { width: 375px; }
  pre { margin: 0; flex: 1; background: #fff; border: 1px solid #d6d0de; border-radius: 8px; padding: 20px; white-space: pre-wrap; font: 13px/1.5 ui-monospace, monospace; }
  .err { background: #fdecec; color: #8a1c1c; border-radius: 8px; padding: 12px 16px; }
  .hint { color: #6b6474; font-size: 12px; }
</style>
</head>
<body>
<nav>
  <h1>Email preview</h1>
  {{range .Nav}}<a href="{{$.Base}}?email={{.ID}}"{{if .Active}} class="active"{{end}}>{{.Label}}<small>{{.Subject}}</small></a>
  {{end}}
</nav>
<main>
  <header>
    <dl>
      <dt>Subject</dt><dd>{{.Subject}}</dd>
      <dt>To</dt><dd>{{.To}}</dd>
    </dl>
    <div class="toggles">
      <span class="toggle">
        <a href="{{.Base}}?email={{.ID}}{{if .Phone}}&width=phone{{end}}"{{if not .TextView}} class="on"{{end}}>HTML</a>
        <a href="{{.Base}}?email={{.ID}}&view=text{{if .Phone}}&width=phone{{end}}"{{if .TextView}} class="on"{{end}}>Text</a>
      </span>
      {{if not .TextView}}<span class="toggle">
        <a href="{{.Base}}?email={{.ID}}"{{if not .Phone}} class="on"{{end}}>Desktop</a>
        <a href="{{.Base}}?email={{.ID}}&width=phone"{{if .Phone}} class="on"{{end}}>Phone</a>
      </span>{{end}}
    </div>
  </header>
  {{if .Err}}<div class="err">{{.Err}}</div>
  {{else if .TextView}}<pre>{{.Text}}</pre>
  {{else}}<div class="frame"><iframe src="{{.Base}}/{{.ID}}.html"{{if .Phone}} class="phone"{{end}} title="{{.Subject}}"></iframe></div>{{end}}
  <p class="hint">Edit templates in internal/infrastructure/email/templates/. This page reloads when the backend restarts.</p>
</main>
<script>
  const build = {{.Build}};
  setInterval(async () => {
    try {
      const res = await fetch({{.Base}} + "/build", { cache: "no-store" });
      if (res.ok && (await res.text()) !== build) location.reload();
    } catch {
      // The backend is restarting; try again.
    }
  }, 1000);
</script>
</body>
</html>
`))
