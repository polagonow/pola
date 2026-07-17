package actions

import (
	"context"
	"time"

	"saas-starter/db/models"
	"saas-starter/lib/reqctx"
	"saas-starter/lib/session"
	"saas-starter/repositories"
)

// ActionResult is the shape every mutating action returns to the client. It
// mirrors the Next.js starter's useActionState state: an inline error or success
// message, plus an optional client-side redirect target.
type ActionResult struct {
	Error    string `json:"error,omitempty"`
	Success  string `json:"success,omitempty"`
	Redirect string `json:"redirect,omitempty"`
}

// Activity-log action names (mirror the starter's ActivityType enum).
const (
	ActivitySignUp           = "SIGN_UP"
	ActivitySignIn           = "SIGN_IN"
	ActivitySignOut          = "SIGN_OUT"
	ActivityUpdatePassword   = "UPDATE_PASSWORD"
	ActivityDeleteAccount    = "DELETE_ACCOUNT"
	ActivityUpdateAccount    = "UPDATE_ACCOUNT"
	ActivityCreateTeam       = "CREATE_TEAM"
	ActivityRemoveTeamMember = "REMOVE_TEAM_MEMBER"
	ActivityInviteTeamMember = "INVITE_TEAM_MEMBER"
	ActivityAcceptInvitation = "ACCEPT_INVITATION"
)

// setSession signs the user id into the JWT session cookie.
func setSession(ctx context.Context, userID uint) { session.Set(ctx, userID) }

// clearSession removes the JWT session cookie (sign-out).
func clearSession(ctx context.Context) { session.Clear(ctx) }

// sessionUserID reads the authenticated user id from the JWT session.
func sessionUserID(ctx context.Context) (uint, bool) { return session.UserID(ctx) }

// logActivity records a team activity entry, best-effort (errors are ignored so
// logging never blocks the primary action).
func logActivity(ctx context.Context, repo repositories.ActivityLogRepository, teamID uint, userID uint, action string) {
	uid := userID
	entry := &models.ActivityLog{
		TeamID:    teamID,
		UserID:    &uid,
		Action:    action,
		Timestamp: time.Now(),
	}
	if ip := reqctx.IP(ctx); ip != "" {
		entry.IPAddress = &ip
	}
	_ = repo.Create(ctx, entry)
}
