package mailer

import (
	"github.com/polagonow/pola/core"
)

// Config holds mailer configuration, typically parsed from Polafile.hcl.
type Config struct {
	From string // default sender address
}

// Plugin returns the mailer plugin. It resolves the EmailRenderer and
// MailTransport from the DI container (registered by renderer/transport
// plugins) and provides a BaseFactory for constructing mailer Base values.
func Plugin(cfg Config) core.Plugin {
	return &mailerPlugin{cfg: cfg}
}

type mailerPlugin struct {
	cfg Config
}

func (p *mailerPlugin) Name() string { return "mailer" }

func (p *mailerPlugin) Register(r *core.Registry) {
	renderer := core.MustInvoke[EmailRenderer](r)
	transport := core.MustInvoke[core.MailTransport](r)
	logger, _ := core.Invoke[core.Logger](r)

	factory := &BaseFactory{
		renderer:  renderer,
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
