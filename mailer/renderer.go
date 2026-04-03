package mailer

import "context"

// EmailRenderer renders email templates to HTML and plain text strings.
type EmailRenderer interface {
	RenderEmail(ctx context.Context, template, layout string, props map[string]any) (html, text string, err error)
}

// TemplateLoader is optionally implemented by renderers that load templates
// from the filesystem (as opposed to receiving a compiled JS bundle).
type TemplateLoader interface {
	LoadTemplates(mailersDir string) error
}
