// Package mailer provides an ActionMailer-style email composition and delivery
// system. Mailers are Go structs that embed Base and define methods which
// compose Message values. Rendering is delegated to react.email templates
// executed in the JS engine; delivery is handled by a pluggable MailTransport.
package mailer

import (
	"context"
	"fmt"

	"github.com/polagonow/pola/core"
)

// Defaults holds per-mailer default values that are merged into every Message
// unless explicitly overridden by an option.
type Defaults struct {
	From   string
	Layout string // layout name under app/mailers/layouts/; default "default"
}

// Base is embedded in user-defined mailer structs. It provides Mail() for
// composing messages and Deliver() for rendering + sending.
type Base struct {
	renderer  EmailRenderer
	transport core.MailTransport
	logger    core.Logger
	defaults  Defaults
}

// NewBase creates a Base with the given dependencies.
func NewBase(renderer EmailRenderer, transport core.MailTransport, logger core.Logger, defaults Defaults) Base {
	if defaults.Layout == "" {
		defaults.Layout = "default"
	}
	return Base{
		renderer:  renderer,
		transport: transport,
		logger:    logger,
		defaults:  defaults,
	}
}

// Mail composes a Message from the given options, merging with the mailer's defaults.
func (b *Base) Mail(opts ...MessageOption) *Message {
	msg := &Message{
		from:       b.defaults.From,
		layoutName: b.defaults.Layout,
		headers:    make(map[string]string),
	}
	for _, opt := range opts {
		opt(msg)
	}
	return msg
}

// Deliver renders the email template and sends it via the transport.
func (b *Base) Deliver(ctx context.Context, msg *Message) error {
	html, text, err := b.renderer.RenderEmail(ctx, msg.templateName, msg.layoutName, msg.props)
	if err != nil {
		return fmt.Errorf("mailer: render %s: %w", msg.templateName, err)
	}

	envelope := &core.MailMessage{
		From:    msg.from,
		To:      msg.to,
		CC:      msg.cc,
		BCC:     msg.bcc,
		ReplyTo: msg.replyTo,
		Subject: msg.subject,
		HTML:    html,
		Text:    text,
		Headers: msg.headers,
	}

	if err := b.transport.Send(ctx, envelope); err != nil {
		return fmt.Errorf("mailer: send %s: %w", msg.templateName, err)
	}
	return nil
}

// Message represents a composed-but-not-yet-rendered email.
type Message struct {
	to           []string
	cc           []string
	bcc          []string
	from         string
	replyTo      string
	subject      string
	templateName string         // e.g. "user_mailer/welcome"
	layoutName   string         // e.g. "default"
	props        map[string]any // passed to React component as props
	headers      map[string]string
}

// MessageOption configures a Message.
type MessageOption func(*Message)

// To sets the recipient addresses.
func To(addrs ...string) MessageOption {
	return func(m *Message) { m.to = addrs }
}

// CC sets the CC addresses.
func CC(addrs ...string) MessageOption {
	return func(m *Message) { m.cc = addrs }
}

// BCC sets the BCC addresses.
func BCC(addrs ...string) MessageOption {
	return func(m *Message) { m.bcc = addrs }
}

// From overrides the default sender address.
func From(addr string) MessageOption {
	return func(m *Message) { m.from = addr }
}

// ReplyTo sets the reply-to address.
func ReplyTo(addr string) MessageOption {
	return func(m *Message) { m.replyTo = addr }
}

// Subject sets the email subject line.
func Subject(s string) MessageOption {
	return func(m *Message) { m.subject = s }
}

// Template sets the template name and props for rendering.
// The template name is a path relative to app/mailers/ (e.g. "user_mailer/welcome").
func Template(name string, props map[string]any) MessageOption {
	return func(m *Message) {
		m.templateName = name
		m.props = props
	}
}

// Layout overrides the default layout name. Pass "" to disable the layout.
func Layout(name string) MessageOption {
	return func(m *Message) { m.layoutName = name }
}

// Header adds a custom email header.
func Header(key, value string) MessageOption {
	return func(m *Message) { m.headers[key] = value }
}
