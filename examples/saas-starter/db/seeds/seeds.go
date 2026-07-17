// Package seeds populates the database with initial data.
//
// Run it with:  pola db seed
package seeds

import (
	"context"
	"fmt"
	"time"

	"github.com/polagonow/pola/auth/password"
	"github.com/polagonow/pola/core"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// Seed creates a test user (test@test.com / admin123), a team, and an owner
// membership — mirroring the Next.js starter's db/seed.ts. It is idempotent:
// re-running skips creation if the test user already exists.
func Seed(ctx context.Context, r *core.Registry) error {
	users := core.MustInvoke[repositories.UserRepository](r)
	teams := core.MustInvoke[repositories.TeamRepository](r)
	members := core.MustInvoke[repositories.TeamMemberRepository](r)

	const email = "test@test.com"
	if _, err := users.GetByEmail(ctx, email); err == nil {
		fmt.Println("Seed: test user already exists — skipping.")
		return nil
	}

	hash, err := password.Hash("admin123")
	if err != nil {
		return err
	}
	user := &models.User{Name: "Test User", Email: email, PasswordHash: hash, Role: "owner"}
	if err := users.Create(ctx, user); err != nil {
		return err
	}

	team := &models.Team{Name: "Test Team"}
	if err := teams.Create(ctx, team); err != nil {
		return err
	}

	if err := members.Create(ctx, &models.TeamMember{
		UserID: user.ID, TeamID: team.ID, Role: "owner", JoinedAt: time.Now(),
	}); err != nil {
		return err
	}

	fmt.Printf("Seed: created user %s (password: admin123) and team %q.\n", email, team.Name)
	return nil
}
