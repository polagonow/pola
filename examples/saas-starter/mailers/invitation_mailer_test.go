package mailers

import (
	"strings"
	"testing"

	"github.com/polagonow/pola/mailer"
)


func TestInvitationMailer_Invite_BuildsMessage(t *testing.T) {
	m := NewInvitationMailer(mailer.Base{})
	msg := m.Invite()
	if msg == nil {
		t.Fatal("expected non-nil *mailer.Message")
	}
	// The template path follows the convention "<name>_mailer/<action>".
	want := "invitation_mailer/invite"
	if !strings.Contains(msg.TemplateName(), want) {
		t.Fatalf("template: got %q, want it to contain %q", msg.TemplateName(), want)
	}
	if msg.SubjectLine() == "" {
		t.Fatal("expected a non-empty subject")
	}
	if len(msg.Recipients()) == 0 {
		t.Fatal("expected at least one recipient")
	}
}

