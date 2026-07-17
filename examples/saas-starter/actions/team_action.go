package actions

import (
	"context"
	"strings"
	"time"

	"github.com/polagonow/pola/core"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// TeamAction implements team membership management (RBAC: owner-only mutations).
// Bridged to React as `TeamAction`.
type TeamAction struct {
	users    repositories.UserRepository
	teams    repositories.TeamRepository
	members  repositories.TeamMemberRepository
	invites  repositories.InvitationRepository
	activity repositories.ActivityLogRepository
}

// NewTeamAction resolves the TeamAction's dependencies from the DI registry.
func NewTeamAction(r *core.Registry) *TeamAction {
	return &TeamAction{
		users:    core.MustInvoke[repositories.UserRepository](r),
		teams:    core.MustInvoke[repositories.TeamRepository](r),
		members:  core.MustInvoke[repositories.TeamMemberRepository](r),
		invites:  core.MustInvoke[repositories.InvitationRepository](r),
		activity: core.MustInvoke[repositories.ActivityLogRepository](r),
	}
}

// RemoveTeamMember removes a member from the caller's team (owner-only).
func (t *TeamAction) RemoveTeamMember(ctx context.Context, memberID uint) (*ActionResult, error) {
	caller, ok := t.ownerMembership(ctx)
	if !ok {
		return &ActionResult{Error: "Only team owners can remove members."}, nil
	}
	target, err := t.members.Get(ctx, memberID)
	if err != nil {
		return &ActionResult{Error: "Member not found."}, nil
	}
	if target.TeamID != caller.TeamID {
		return &ActionResult{Error: "That member is not on your team."}, nil
	}
	if err := t.members.Delete(ctx, memberID); err != nil {
		return nil, err
	}
	logActivity(ctx, t.activity, caller.TeamID, caller.UserID, ActivityRemoveTeamMember)
	return &ActionResult{Success: "Member removed."}, nil
}

// InviteTeamMember invites a new member by email (owner-only) and sends the
// invitation email.
func (t *TeamAction) InviteTeamMember(ctx context.Context, email, role string) (*ActionResult, error) {
	caller, ok := t.ownerMembership(ctx)
	if !ok {
		return &ActionResult{Error: "Only team owners can invite members."}, nil
	}
	email = normalizeEmail(email)
	if role != "owner" && role != "member" {
		role = "member"
	}

	// Already a member?
	if u, err := t.users.GetByEmail(ctx, email); err == nil {
		if m, err := t.members.GetByUserID(ctx, u.ID); err == nil && m.TeamID == caller.TeamID {
			return &ActionResult{Error: "That user is already on your team."}, nil
		}
	}
	// Already invited?
	if _, err := t.invites.GetPending(ctx, caller.TeamID, email); err == nil {
		return &ActionResult{Error: "An invitation is already pending for this email."}, nil
	}

	if err := t.invites.Create(ctx, &models.Invitation{
		TeamID:      caller.TeamID,
		Email:       email,
		Role:        role,
		InvitedByID: caller.UserID,
		InvitedAt:   time.Now(),
		Status:      "pending",
	}); err != nil {
		return nil, err
	}
	logActivity(ctx, t.activity, caller.TeamID, caller.UserID, ActivityInviteTeamMember)

	// The starter leaves email delivery as a follow-up. A generated mailer lives
	// in mailers/invitation_mailer.go (see `pola generate mailer`); wire it up by
	// adding a mailer {} block to Polafile.hcl and injecting *mailers.InvitationMailer.
	return &ActionResult{Success: "Invitation sent."}, nil
}

// ownerMembership returns the caller's membership when they are an owner.
func (t *TeamAction) ownerMembership(ctx context.Context) (*models.TeamMember, bool) {
	uid, ok := sessionUserID(ctx)
	if !ok {
		return nil, false
	}
	m, err := t.members.GetByUserID(ctx, uid)
	if err != nil || !strings.EqualFold(m.Role, "owner") {
		return nil, false
	}
	return m, true
}
