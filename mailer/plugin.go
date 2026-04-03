package mailer

import (
	"github.com/polagonow/pola/core"
)

// Config holds mailer configuration, typically parsed from Polafile.hcl.
type Config struct {
	Transport string // "smtp", "log"; default "log"
	From      string // default sender address
	SMTP      SMTPConfig
}

// Plugin returns the mailer plugin. It registers the MailTransport in the
// DI container and provides a BaseFactory for constructing mailer Base values.
func Plugin(cfg Config) core.Plugin {
	return &mailerPlugin{cfg: cfg}
}

type mailerPlugin struct {
	cfg      Config
	renderer *ReactEmailRenderer
}

func (p *mailerPlugin) Name() string { return "mailer" }

func (p *mailerPlugin) Register(r *core.Registry) {
	// Resolve dependencies.
	engine := core.MustInvoke[core.JSEngine](r)
	logger, _ := core.Invoke[core.Logger](r)

	// Create the email renderer.
	p.renderer = NewReactEmailRenderer(engine, logger)

	// Select transport.
	var transport core.MailTransport
	switch p.cfg.Transport {
	case "smtp":
		transport = NewSMTPTransport(p.cfg.SMTP)
	default:
		transport = NewLogTransport(logger)
	}

	core.ProvideValue[core.MailTransport](r, transport)

	// Register the email renderer so the pipeline can load the email bundle.
	core.ProvideValue[*ReactEmailRenderer](r, p.renderer)

	// Register a BaseFactory so user mailers can construct Base values.
	factory := &BaseFactory{
		renderer:  p.renderer,
		transport: transport,
		logger:    logger,
		from:      p.cfg.From,
	}
	core.ProvideValue[*BaseFactory](r, factory)
}

// BaseFactory creates Base values for user-defined mailer structs.
// Registered in the DI container by the mailer plugin.
type BaseFactory struct {
	renderer  EmailRenderer
	transport core.MailTransport
	logger    core.Logger
	from      string
}

// NewBase creates a Base with the factory's dependencies and the given defaults.
func (f *BaseFactory) NewBase(defaults Defaults) Base {
	if defaults.From == "" {
		defaults.From = f.from
	}
	return NewBase(f.renderer, f.transport, f.logger, defaults)
}
