package mailers

import (
	"github.com/polagonow/pola/mailer"
)

// InvitationMailer sends emails related to invitation operations.
type InvitationMailer struct {
	mailer.Base
}

// NewInvitationMailer creates a new InvitationMailer with injected dependencies.
func NewInvitationMailer(base mailer.Base) *InvitationMailer {
	return &InvitationMailer{Base: base}
}

// Invite composes the invite email.
func (m *InvitationMailer) Invite() *mailer.Message {
	return m.Mail(
		mailer.To("recipient@example.com"),
		mailer.Subject("Invite"),
		mailer.Template("invitation_mailer/invite", map[string]any{
			// TODO: add template props
		}),
	)
}
