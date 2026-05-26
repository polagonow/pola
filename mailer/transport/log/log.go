package log

import (
	"context"

	"github.com/polagonow/pola/core"
)

// Transport logs the email to the framework logger instead of sending.
// Useful for development and testing.
type Transport struct {
	logger core.Logger
}

// Plugin returns a core.Plugin that registers a log-based mail transport.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "mailer.transport.log",
		Fn: func(r *core.Registry) {
			logger, _ := core.Invoke[core.Logger](r)
			core.ProvideValue[core.MailTransport](r, &Transport{logger: logger})
		},
	}
}

func (t *Transport) Name() string { return "log" }

func (t *Transport) Send(_ context.Context, msg *core.MailMessage) error {
	t.logger.Info("mailer: email delivered",
		"from", msg.From,
		"to", msg.To,
		"subject", msg.Subject,
		"html_length", len(msg.HTML),
		"text_length", len(msg.Text),
	)
	return nil
}
