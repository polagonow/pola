package authz_test

import (
	"errors"
	"testing"

	"github.com/polagonow/pola/auth/authz"
)

func TestHasRole(t *testing.T) {
	if !authz.HasRole("owner", "Owner") { // case-insensitive
		t.Fatal("owner should match Owner")
	}
	if authz.HasRole("member", "owner") {
		t.Fatal("member is not owner")
	}
	if !authz.HasRole([]string{"member", "admin"}, "admin") {
		t.Fatal("slice subject should match admin")
	}
	if authz.HasRole("", "owner") {
		t.Fatal("empty subject holds no role")
	}
}

func TestHasAnyAll(t *testing.T) {
	roles := []string{"member", "billing"}
	if !authz.HasAny(roles, "owner", "billing") {
		t.Fatal("should match billing via HasAny")
	}
	if authz.HasAll(roles, "member", "owner") {
		t.Fatal("missing owner → HasAll false")
	}
	if !authz.HasAll(roles, "member", "billing") {
		t.Fatal("has both → HasAll true")
	}
}

func TestRequireRole(t *testing.T) {
	if err := authz.RequireRole("owner", "owner"); err != nil {
		t.Fatalf("owner allowed: %v", err)
	}
	if err := authz.RequireRole("member", "owner"); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("member should be forbidden, got %v", err)
	}
	if err := authz.RequireAny("member", "owner", "member"); err != nil {
		t.Fatalf("RequireAny should pass: %v", err)
	}
}

func TestPolicy(t *testing.T) {
	p := authz.NewPolicy().
		Grant("owner", "team.manage", "billing.manage").
		Grant("member", "team.view")

	if !p.Can("owner", "billing.manage") {
		t.Fatal("owner can manage billing")
	}
	if p.Can("member", "billing.manage") {
		t.Fatal("member cannot manage billing")
	}
	if !p.Can("member", "team.view") {
		t.Fatal("member can view team")
	}
	if err := p.Require("member", "team.manage"); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("member managing team should be forbidden, got %v", err)
	}
	if err := p.Require("owner", "team.manage"); err != nil {
		t.Fatalf("owner managing team allowed: %v", err)
	}
}
