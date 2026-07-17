package actions

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/polagonow/pola/auth/password"
	"github.com/polagonow/pola/core"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// AuthAction implements email/password authentication and account management.
// Bridged to React as `AuthAction` in "@pola/actions".
type AuthAction struct {
	users    repositories.UserRepository
	teams    repositories.TeamRepository
	members  repositories.TeamMemberRepository
	activity repositories.ActivityLogRepository
	invites  repositories.InvitationRepository
}

// NewAuthAction resolves the AuthAction's dependencies from the DI registry.
func NewAuthAction(r *core.Registry) *AuthAction {
	return &AuthAction{
		users:    core.MustInvoke[repositories.UserRepository](r),
		teams:    core.MustInvoke[repositories.TeamRepository](r),
		members:  core.MustInvoke[repositories.TeamMemberRepository](r),
		activity: core.MustInvoke[repositories.ActivityLogRepository](r),
		invites:  core.MustInvoke[repositories.InvitationRepository](r),
	}
}

// SignIn verifies credentials, sets the session, and logs SIGN_IN.
func (a *AuthAction) SignIn(ctx context.Context, email, pass string) (*ActionResult, error) {
	email = normalizeEmail(email)
	user, err := a.users.GetByEmail(ctx, email)
	if err != nil {
		return &ActionResult{Error: "Invalid email or password."}, nil
	}
	if ok, _ := password.Verify(pass, user.PasswordHash); !ok {
		return &ActionResult{Error: "Invalid email or password."}, nil
	}
	setSession(ctx, user.ID)
	if team, err := a.teams.GetForUser(ctx, user.ID); err == nil {
		logActivity(ctx, a.activity, team.ID, user.ID, ActivitySignIn)
	}
	return &ActionResult{Redirect: "/dashboard"}, nil
}

// SignUp creates a user and either accepts an invitation (joining that team) or
// creates a fresh team, then sets the session. Logs SIGN_UP plus CREATE_TEAM or
// ACCEPT_INVITATION.
func (a *AuthAction) SignUp(ctx context.Context, email, pass, inviteID string) (*ActionResult, error) {
	email = normalizeEmail(email)
	if !strings.Contains(email, "@") || len(pass) < 8 {
		return &ActionResult{Error: "Enter a valid email and a password of at least 8 characters."}, nil
	}
	if _, err := a.users.GetByEmail(ctx, email); err == nil {
		return &ActionResult{Error: "A user with this email already exists."}, nil
	}

	hash, err := password.Hash(pass)
	if err != nil {
		return nil, err
	}
	user := &models.User{Email: email, PasswordHash: hash, Role: "member"}
	if err := a.users.Create(ctx, user); err != nil {
		return nil, err
	}

	var team *models.Team
	memberRole := "owner"

	if inviteID != "" {
		if id, convErr := strconv.Atoi(inviteID); convErr == nil {
			if inv, err := a.invites.Get(ctx, uint(id)); err == nil &&
				inv.Status == "pending" && strings.EqualFold(inv.Email, email) {
				if t, err := a.teams.Get(ctx, inv.TeamID); err == nil {
					team = t
					memberRole = inv.Role
					inv.Status = "accepted"
					_ = a.invites.Update(ctx, inv)
					logActivity(ctx, a.activity, team.ID, user.ID, ActivityAcceptInvitation)
				}
			}
		}
	}

	if team == nil {
		team = &models.Team{Name: email + "'s Team"}
		if err := a.teams.Create(ctx, team); err != nil {
			return nil, err
		}
		memberRole = "owner"
		logActivity(ctx, a.activity, team.ID, user.ID, ActivityCreateTeam)
	}

	if err := a.members.Create(ctx, &models.TeamMember{
		UserID: user.ID, TeamID: team.ID, Role: memberRole, JoinedAt: time.Now(),
	}); err != nil {
		return nil, err
	}

	logActivity(ctx, a.activity, team.ID, user.ID, ActivitySignUp)
	setSession(ctx, user.ID)
	return &ActionResult{Redirect: "/dashboard"}, nil
}

// SignOut logs SIGN_OUT and clears the session.
func (a *AuthAction) SignOut(ctx context.Context) (*ActionResult, error) {
	if uid, ok := sessionUserID(ctx); ok {
		if team, err := a.teams.GetForUser(ctx, uid); err == nil {
			logActivity(ctx, a.activity, team.ID, uid, ActivitySignOut)
		}
	}
	clearSession(ctx)
	return &ActionResult{Redirect: "/sign-in"}, nil
}

// UpdatePassword changes the current user's password.
func (a *AuthAction) UpdatePassword(ctx context.Context, current, newPass, confirm string) (*ActionResult, error) {
	uid, ok := sessionUserID(ctx)
	if !ok {
		return &ActionResult{Error: "Not authenticated."}, nil
	}
	user, err := a.users.Get(ctx, uid)
	if err != nil {
		return nil, err
	}
	if ok, _ := password.Verify(current, user.PasswordHash); !ok {
		return &ActionResult{Error: "Current password is incorrect."}, nil
	}
	if current == newPass {
		return &ActionResult{Error: "New password must be different from the current password."}, nil
	}
	if len(newPass) < 8 {
		return &ActionResult{Error: "Password must be at least 8 characters."}, nil
	}
	if newPass != confirm {
		return &ActionResult{Error: "Passwords do not match."}, nil
	}
	hash, err := password.Hash(newPass)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = hash
	if err := a.users.Update(ctx, user); err != nil {
		return nil, err
	}
	if team, err := a.teams.GetForUser(ctx, uid); err == nil {
		logActivity(ctx, a.activity, team.ID, uid, ActivityUpdatePassword)
	}
	return &ActionResult{Success: "Password updated."}, nil
}

// UpdateAccount changes the current user's name and email.
func (a *AuthAction) UpdateAccount(ctx context.Context, name, email string) (*ActionResult, error) {
	uid, ok := sessionUserID(ctx)
	if !ok {
		return &ActionResult{Error: "Not authenticated."}, nil
	}
	user, err := a.users.Get(ctx, uid)
	if err != nil {
		return nil, err
	}
	user.Name = name
	user.Email = normalizeEmail(email)
	if err := a.users.Update(ctx, user); err != nil {
		return nil, err
	}
	if team, err := a.teams.GetForUser(ctx, uid); err == nil {
		logActivity(ctx, a.activity, team.ID, uid, ActivityUpdateAccount)
	}
	return &ActionResult{Success: "Account updated."}, nil
}

// DeleteAccount verifies the password, soft-deletes the user (obfuscating the
// email so it can be reused), removes their membership, and clears the session.
func (a *AuthAction) DeleteAccount(ctx context.Context, pass string) (*ActionResult, error) {
	uid, ok := sessionUserID(ctx)
	if !ok {
		return &ActionResult{Error: "Not authenticated."}, nil
	}
	user, err := a.users.Get(ctx, uid)
	if err != nil {
		return nil, err
	}
	if ok, _ := password.Verify(pass, user.PasswordHash); !ok {
		return &ActionResult{Error: "Incorrect password."}, nil
	}
	if team, err := a.teams.GetForUser(ctx, uid); err == nil {
		logActivity(ctx, a.activity, team.ID, uid, ActivityDeleteAccount)
	}
	if m, err := a.members.GetByUserID(ctx, uid); err == nil {
		_ = a.members.Delete(ctx, m.ID)
	}
	// Obfuscate the email so it can be re-registered, then soft-delete.
	user.Email = fmt.Sprintf("%s-%d-deleted", user.Email, user.ID)
	_ = a.users.Update(ctx, user)
	_ = a.users.Delete(ctx, user.ID)
	clearSession(ctx)
	return &ActionResult{Redirect: "/sign-in"}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
