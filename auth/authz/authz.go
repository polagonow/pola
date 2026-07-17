// Package authz provides small, dependency-free authorization primitives —
// role checks and an optional role→permission policy — usable from both bridged
// actions and route handlers. It is deliberately unopinionated about where a
// subject's roles come from (session claim, DB row, etc.): you pass them in.
//
//	// In an action, after loading the current user's role:
//	if err := authz.RequireRole(user.Role, "owner"); err != nil {
//	    return &ActionResult{Error: "Owners only."}, nil
//	}
//
//	// Or with a permission policy:
//	policy := authz.NewPolicy().
//	    Grant("owner", "team.manage", "billing.manage").
//	    Grant("member", "team.view")
//	if !policy.Can(user.Role, "billing.manage") { ... }
package authz

import (
	"errors"
	"strings"
)

// ErrForbidden is returned by the Require* helpers when a check fails.
var ErrForbidden = errors.New("authz: forbidden")

// HasRole reports whether the subject holds want. The subject's role(s) may be a
// single role ("owner") or several (["owner","admin"]); matching is
// case-insensitive.
func HasRole(subject any, want string) bool {
	for _, r := range rolesOf(subject) {
		if strings.EqualFold(r, want) {
			return true
		}
	}
	return false
}

// HasAny reports whether the subject holds at least one of wants.
func HasAny(subject any, wants ...string) bool {
	for _, w := range wants {
		if HasRole(subject, w) {
			return true
		}
	}
	return false
}

// HasAll reports whether the subject holds every one of wants.
func HasAll(subject any, wants ...string) bool {
	for _, w := range wants {
		if !HasRole(subject, w) {
			return false
		}
	}
	return len(wants) > 0
}

// RequireRole returns ErrForbidden unless the subject holds want.
func RequireRole(subject any, want string) error {
	if HasRole(subject, want) {
		return nil
	}
	return ErrForbidden
}

// RequireAny returns ErrForbidden unless the subject holds at least one of wants.
func RequireAny(subject any, wants ...string) error {
	if HasAny(subject, wants...) {
		return nil
	}
	return ErrForbidden
}

// Policy maps roles to the permissions they grant. It is safe to build once at
// startup and read concurrently (no mutation after construction).
type Policy struct {
	grants map[string]map[string]bool
}

// NewPolicy creates an empty policy.
func NewPolicy() *Policy {
	return &Policy{grants: map[string]map[string]bool{}}
}

// Grant gives role the listed permissions and returns the policy for chaining.
func (p *Policy) Grant(role string, perms ...string) *Policy {
	role = strings.ToLower(role)
	set := p.grants[role]
	if set == nil {
		set = map[string]bool{}
		p.grants[role] = set
	}
	for _, perm := range perms {
		set[perm] = true
	}
	return p
}

// Can reports whether any of the subject's roles grants perm.
func (p *Policy) Can(subject any, perm string) bool {
	for _, r := range rolesOf(subject) {
		if p.grants[strings.ToLower(r)][perm] {
			return true
		}
	}
	return false
}

// Require returns ErrForbidden unless the subject may perform perm.
func (p *Policy) Require(subject any, perm string) error {
	if p.Can(subject, perm) {
		return nil
	}
	return ErrForbidden
}

// rolesOf normalizes a subject's roles from a string or []string.
func rolesOf(subject any) []string {
	switch v := subject.(type) {
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	case []string:
		return v
	default:
		return nil
	}
}
