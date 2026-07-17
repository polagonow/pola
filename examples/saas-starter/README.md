# SaaS Starter (Pola port)

A faithful port of the [Next.js SaaS Starter](https://github.com/nextjs/saas-starter)
to the **Pola** framework — built almost entirely with the `pola` CLI generators.

Same feature set as the original, on a Go backend that renders React Server
Components:

- Email/password auth with a **JWT session cookie** (HS256, `AUTH_SECRET`, 1-day, HttpOnly/Secure/SameSite=Lax)
- Teams / multi-tenancy, **RBAC** (owner/member)
- Activity logging (SIGN_IN, SIGN_UP, CREATE_TEAM, UPDATE_PASSWORD, …)
- Team invitations
- **Stripe** subscriptions: checkout with a 14-day trial, customer portal, webhook-driven sync
- Dashboard (account, security, activity, team) + pricing + marketing landing
- GORM + SQLite (dev) / Postgres (prod), auto-migrations
- shadcn/ui + Tailwind v4

## Tech mapping vs. the original

| Next.js starter | This port (Pola) |
|---|---|
| Server Actions (`'use server'`) | Go **actions** in `actions/` bridged to `@pola/actions`, wrapped by `web/lib/actions.ts` |
| `jose` JWT cookie | pola **`middleware/jwt`** plugin (`jwt {}` in Polafile) |
| bcrypt (`SALT_ROUNDS=10`) | `auth/password` (bcrypt DefaultCost = 10) |
| Drizzle + Postgres | GORM models in `db/models/`, SQL migrations, `pola db migrate` |
| `stripe` npm SDK | `github.com/stripe/stripe-go/v79` in `lib/stripe/` |
| `middleware.ts` route guard | `plugins.go` auth-guard middleware |
| `db/seed.ts` | `db/seeds/seeds.go`, run via **`pola db seed`** |
| SWR `/api/user`, `/api/team` | Go routes `routes/apis/...` (`GET /api/user`, `/api/team`, `/api/activity`) |

## Framework features added to Pola for this port

This example drove three additions to the framework itself (rather than working
around gaps in the app):

1. **JWT plugin** — `auth/jwt` (Sign/Verify) + `middleware/jwt` (cookie session, `Set/Get/Clear`) + a first-class `jwt {}` Polafile block.
2. **CSRF path exceptions** — `csrf.WithExempt(...)` + `csrf { exempt = [...] }`, so the Stripe webhook (which authenticates via its own signature) bypasses CSRF while the rest of the app stays protected. The bridge client also now sends the CSRF token automatically.
3. **Rails-style seeds** — a `db/seeds` package + `pola generate seed` + `pola db seed`.

---

## Run it

```bash
# 1. Build the pola CLI from the repo root (it includes the features above)
cd ../..            # repo root
go build -o /usr/local/bin/pola ./cmd/pola   # or: go run ./cmd/pola <args>
cd examples/saas-starter

# 2. Configure environment
cp .env.example .env
# edit .env: set AUTH_SECRET (openssl rand -hex 32) and, for billing, your
# Stripe test keys (STRIPE_SECRET_KEY, STRIPE_WEBHOOK_SECRET).

# 3. Install web deps, migrate, seed, run
(cd web && pnpm install)
pola db migrate       # creates dev.db + all tables
pola db seed          # creates test@test.com / admin123 + a team
pola dev              # http://localhost:3000
```

Sign in with **test@test.com** / **admin123**, or sign up to create a new team.

### Stripe (test mode)

```bash
stripe listen --forward-to localhost:3000/api/stripe/webhook
# copy the printed whsec_... into .env as STRIPE_WEBHOOK_SECRET
```

Then open `/pricing`, start a subscription (test card `4242 4242 4242 4242`), and
Stripe redirects back to `/api/stripe/checkout`, which syncs the subscription onto
the team and redirects to `/dashboard`.

---

## Reproduce from scratch (the CLI commands)

Everything structural was generated with the `pola` CLI; only business logic and
the bespoke UI were authored on top. Run these from `examples/`:

```bash
# --- Scaffold the full-stack app (shadcn + tailwind + react) ---
pola new saas-starter \
  --renderer react --bundler esbuild --router nextjs \
  --css tailwind --ui shadcn --vm goja --framework std \
  --pm pnpm --pola-path ../..
cd saas-starter
# (pola new prompts for the app name — accept the default.)

# --- Configure the Polafile (add database, jwt, csrf-exempt blocks) ---
# Edit Polafile.hcl to add:
#   jwt      { cookie = "session"  expiry = "24h"  secret_env = "AUTH_SECRET" }
#   csrf     { exempt = ["/api/stripe/webhook"] }
#   database { orm = "gorm"  adapter = "sqlite"  models = "db/models"
#              migrations { directory = "db/migrations"  format = "sql" }
#              env "production" { adapter = "postgresql" } }

# --- Models + repositories + services + migrations ---
pola generate model      User name:string email:email:uniq password_hash:string role:string --soft-delete
pola generate repository User name:string email:email:uniq password_hash:string role:string
pola generate service    User name:string email:email:uniq password_hash:string role:string
pola generate scaffold Team name:string stripe_customer_id:string:uniq stripe_subscription_id:string:uniq stripe_product_id:string plan_name:string subscription_status:string --skip-action --skip-route --skip-views --skip-zod
pola generate scaffold TeamMember  'user:references{User}' 'team:references{Team}' role:string joined_at:time            --skip-action --skip-route --skip-views --skip-zod
pola generate scaffold ActivityLog 'team:references{Team}' 'user:references{User}' action:string timestamp:time ip_address:string --skip-action --skip-route --skip-views --skip-zod
pola generate scaffold Invitation  'team:references{Team}' email:email 'invited_by:references{User}' role:string invited_at:time status:string --skip-action --skip-route --skip-views --skip-zod

# --- Actions (bridged to @pola/actions) ---
pola generate action Auth     --service User
pola generate action Team     --service Team
pola generate action Payments --service Team

# --- REST routes (SWR endpoints + Stripe) ---
pola generate route api/user   --service User
pola generate route api/team   --service Team
pola generate route api/stripe/checkout GET
pola generate route api/stripe/webhook  POST
# plus routes/apis/activity (GET /api/activity), authored

# --- Mailer + database seeder ---
pola generate mailer Invitation invite
pola generate seed

# --- Stripe SDK ---
go get github.com/stripe/stripe-go/v79
```

After generating, the following were **authored** on top of the CLI skeletons
(business logic the generators can't produce):

- `db/models/*.go` — refined nullability / JSON tags (Stripe fields nullable, `password_hash` hidden)
- `repositories/*_repository.go` + `repositories/gorm/*_queries.go` — custom queries (`GetByEmail`, `GetForUser`, `GetByStripeCustomerID`, `ListForUser`, `GetPending`, …)
- `actions/{auth,team,payments}_action.go` + `actions/helpers.go` — auth, RBAC, activity logging, Stripe
- `routes/apis/**/route.go` — `Path()` overrides + session-aware handlers
- `lib/stripe/stripe.go`, `lib/session/`, `lib/reqctx/` — Stripe, JWT-session helpers, client-IP middleware
- `plugins.go` — auth-guard + client-IP middleware
- `db/seeds/seeds.go` — the seed data
- `web/**` — the React UI (marketing/login/dashboard/pricing) on the generated shadcn components

## Project layout

```
Polafile.hcl            main.go            plugins.go        .env.example
db/models/              db/migrations/     db/seeds/
repositories/  (+ gorm/) services/         actions/          mailers/
lib/{stripe,session,reqctx}/
routes/apis/{user,team,activity,stripe/{checkout,webhook}}/
web/  (app/, components/ui/, lib/)
```
