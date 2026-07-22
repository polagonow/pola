---
name: add-mailer-transport
description: Add a new mail transport (delivery backend) or email template renderer to the Pola mailer subsystem. Use when asked to add, implement, or support a mail delivery service (SES, SendGrid, Resend, Mailgun, ...) or an email template renderer (MJML, Markdown, plain HTML, ...).
---

The mailer is three cooperating plugins wired through DI (`core.Registry`):

- a **transport** plugin provides `core.MailTransport` (delivery),
- a **renderer** plugin provides `mailer.EmailRenderer` (template → HTML/text),
- `mailer.Plugin(mailer.Config{From: ...})` (`mailer/plugin.go`) resolves both
  via `core.MustInvoke` (required) and provides the `*mailer.BaseFactory`
  used by app mailer structs.

The package directory name **is** the Polafile value: generated apps import
`.../mailer/transport/{{.MailerTransport}}` and
`.../mailer/renderer/{{.MailerRenderer}}` directly (see
`internal/autoload/pluginimports/_templates/plugins_go.tmpl`).

## Files to create

| File | Purpose |
|------|---------|
| `mailer/transport/<name>/<name>.go` | Transport impl + `Plugin()` |
| `mailer/renderer/<name>/<name>.go` | Renderer impl + `Plugin()` (renderer only) |

## Part A — Mail transport

### Step 1 — Implement `core.MailTransport`

The interface (`core/interfaces.go`):

```go
type MailTransport interface {
	Name() string
	Send(ctx context.Context, msg *MailMessage) error
}
```

`core.MailMessage` arrives fully rendered: `From`, `To`/`CC`/`BCC []string`,
`ReplyTo`, `Subject`, `HTML`, `Text`, `Headers map[string]string`.

**`mailer/transport/<name>/<name>.go`** — model on `mailer/transport/log/log.go`:

```go
package sendgrid // package name = directory name = Polafile transport value

import (
	"context"

	"github.com/polagonow/pola/core"
)

type Transport struct {
	logger core.Logger
}

// Plugin returns a core.Plugin that registers this mail transport.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "mailer.transport.sendgrid",
		Fn: func(r *core.Registry) {
			logger, _ := core.Invoke[core.Logger](r) // optional dependency
			core.ProvideValue[core.MailTransport](r, &Transport{logger: logger})
		},
	}
}

func (t *Transport) Name() string { return "sendgrid" }

func (t *Transport) Send(ctx context.Context, msg *core.MailMessage) error {
	// Deliver msg. Wrap errors: fmt.Errorf("sendgrid: send: %w", err)
	return nil
}
```

Resolve credentials from env at send time — see `resolveConfig` in
`mailer/transport/smtp/smtp.go`, which does
`cmp.Or(os.Getenv("SMTP_HOST"), t.cfg.Host)` per field.

### Step 2 — Generated-app wiring caveat

`plugins_go.tmpl` invokes the transport plugin like this:

```
{{- if eq .MailerTransport "smtp"}}
	mailertransport.Plugin(mailertransport.Config{ /* SMTP fields */ }),
{{- else}}
	mailertransport.Plugin(),
{{- end}}
```

Only `smtp` receives a `Config` literal. A new transport must therefore export
a **no-arg** `Plugin()` (read config from env vars), or you must add a branch
to `internal/autoload/pluginimports/_templates/plugins_go.tmpl` and thread new
fields through `PluginOpts` + `ApplyMailerOpts` in
`internal/autoload/autoload.go` and the `Mailer` struct in
`polafile/polafile.go`.

## Part B — Email renderer

The interfaces (`mailer/renderer.go`):

```go
type EmailRenderer interface {
	RenderEmail(ctx context.Context, template, layout string, props map[string]any) (html, text string, err error)
}

// Optional — implement when templates load from the filesystem:
type TemplateLoader interface {
	LoadTemplates(mailersDir string) error
}
```

**`mailer/renderer/<name>/<name>.go>`** — model on `mailer/renderer/tmpl/tmpl.go`:

```go
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "mailer.renderer.<name>",
		Fn: func(r *core.Registry) {
			logger, _ := core.Invoke[core.Logger](r)
			renderer := &Renderer{logger: logger}
			core.ProvideValue[mailer.EmailRenderer](r, renderer)
			core.ProvideValue[mailer.TemplateLoader](r, renderer) // only if implemented
			core.ProvideValue(r, renderer) // concrete type, for direct resolution
		},
	}
}
```

Notes:

- `template` names look like `"user_mailer/welcome"`, `layout` like `"default"`.
- Returning `text == ""` is fine — the SMTP transport falls back to
  single-part MIME.
- The `react` renderer (`mailer/renderer/react/react.go`) skips
  `TemplateLoader` and instead exposes `LoadBundle(bundle []byte)` for a
  compiled JS bundle; only use that pattern for JS-based renderers.

## How it's enabled

`Polafile.hcl` (schema in `polafile/polafile.go`; defaults renderer `"react"`,
transport `"log"` — see `ApplyMailerOpts` in `internal/autoload/autoload.go`):

```hcl
mailer {
  renderer  = "tmpl"      # -> mailer/renderer/tmpl
  transport = "sendgrid"  # -> mailer/transport/sendgrid
  from      = "noreply@example.com"

  env "production" {
    transport = "smtp"
  }
}
```

## Verify

```
go build ./...
go test ./mailer/...
```

Then set the Polafile block in an example app and run `pola dev`; the `log`
transport's "mailer: email delivered" output is the reference behavior.
