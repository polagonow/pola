package main

// showcase.go collects short, self-contained snippets for the borrowed-from-
// Goyave improvements that live outside the live CRUD request path. They compile
// as part of this example (functions are intentionally uncalled) so every API
// stays honest and copy-pasteable. The live demonstrations are:
//   • DTOs + partial-update + DTO-flow routes + neutral 404 → routes/products
//   • CORS + health/readiness                               → main.go
//   • query-driven filtering/sorting/field-select           → routes/products GET

import (
	"context"
	"errors"
	"net/http"
	"time"

	"features-showcase/db/models"
	"features-showcase/dto"
	"features-showcase/repositories"

	"github.com/polagonow/pola/auth"
	"github.com/polagonow/pola/core"
	poladto "github.com/polagonow/pola/dto"
	"github.com/polagonow/pola/openapi"
	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"
	"github.com/polagonow/pola/repository/uow"
	"github.com/polagonow/pola/routes"
	"github.com/polagonow/pola/seed"

	"gorm.io/gorm"
)

// --- P0-1: DTOs — partial-update (PATCH) mapping -----------------------------

// applyPartialUpdate writes only the fields the client actually sent: a present
// Optional (even "price": 0) is applied; an omitted field is left untouched.
func applyPartialUpdate(model *models.Product, in dto.UpdateProduct) *models.Product {
	return poladto.Copy(model, in) // ... then repo.Update(ctx, model)
}

// toResponse converts a model to its response DTO, dropping any non-DTO columns.
func toResponse(m *models.Product) dto.Product {
	return poladto.MustConvert[dto.Product](m)
}

// --- P0-3: context-propagated business transactions --------------------------

// atomicWrites runs two repository writes as one unit: both commit, or both roll
// back. Repositories enlist in the ambient transaction automatically.
func atomicWrites(db *gorm.DB, repo repositories.ProductRepository) error {
	var tx uow.TxManager = gormrepo.NewTxManager(db)
	return tx.Transaction(context.Background(), func(ctx context.Context) error {
		if err := repo.Create(ctx, &models.Product{Name: "widget-a"}); err != nil {
			return err // rolls back
		}
		return repo.Create(ctx, &models.Product{Name: "widget-b"}) // commits if nil
	})
}

// --- P0-2: query-driven listing (also used live by the Products GET route) ---

func parseListQuery(raw map[string][]string) repository.ListParams {
	// e.g. ?filter=name||$cont||wid&sort=price,desc&fields=id,name&per_page=5
	return repository.ParseListQuery(raw)
}

// --- P1-4: neutral not-found → HTTP 404 --------------------------------------

func getOr404(c core.Context, m *models.Product, err error) error {
	if err != nil {
		if repository.IsNotFound(err) { // works across gorm/beego/ent
			return core.WithStatus(http.StatusNotFound, err)
		}
		return err
	}
	return c.JSON(http.StatusOK, toResponse(m))
}

// --- P1-8: pluggable authentication + P1-7: per-route middleware -------------

type account struct {
	Email string
	Hash  string
}

type userStore struct{ byEmail map[string]*account }

func (s userStore) FindByUsername(_ context.Context, email string) (*account, error) {
	if u, ok := s.byEmail[email]; ok {
		return u, nil
	}
	return nil, errors.New("no such user")
}

func newAuthenticator(secret []byte) auth.Authenticator[account] {
	return &auth.JWTAuthenticator[account]{
		Users:  userStore{byEmail: map[string]*account{}},
		Secret: secret,
	}
}

// login mints a token after (elsewhere) verifying the password with auth/password.
func login(secret []byte) (string, error) {
	return auth.IssueToken("ada@example.com", secret, 24*time.Hour, nil)
}

// adminRoutes protects every action by declaring per-route middleware; the
// handler reads the injected user with auth.UserFromContext.
type adminRoutes struct {
	authenticator auth.Authenticator[account]
}

func (r *adminRoutes) Middleware() []core.RouteMiddleware {
	return []core.RouteMiddleware{auth.RouteMiddleware(r.authenticator)}
}

func (r *adminRoutes) Meta() map[string]any {
	return map[string]any{"summary": "Admin-only endpoints", "tags": []string{"admin"}}
}

func (r *adminRoutes) GET(c core.Context) error {
	u, ok := auth.UserFromContext[account](c.Ctx())
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	return c.JSON(http.StatusOK, core.M{"you": u.Email})
}

// --- P1-9: OpenAPI 3 spec + SwaggerUI ----------------------------------------

// serveOpenAPI builds an OpenAPI document from discovered routes and returns the
// middleware serving it at /openapi.json and a SwaggerUI page at /openapi.
func serveOpenAPI(specs routes.RouteSpecs) core.Middleware {
	doc := openapi.Generate(specs, openapi.Info{Title: "Features Showcase API", Version: "1.0.0"})
	return openapi.Serve(doc)
}

// --- P1-6: faker-free test/seed factories ------------------------------------

var productFactory = seed.NewFactory(func() *models.Product {
	return &models.Product{Name: "seed-widget", Price: 9.99}
})

func seedProducts(ctx context.Context, repo repositories.ProductRepository, n int) ([]*models.Product, error) {
	return productFactory.Override(func(p *models.Product) { p.Description = "seeded" }).
		Save(ctx, repo.Create, n)
}
