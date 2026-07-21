package products

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"features-showcase/db/models"
	"features-showcase/services"
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	"github.com/polagonow/pola/webframework/std"
)

// newTestRoute wires the route to the given mock service through a fresh DI
// registry, exercising the real registry-style constructor.
func newTestRoute(svc services.ProductServiceInterface) *Route {
	r := core.NewRegistry()
	core.ProvideValue[services.ProductServiceInterface](r, svc)
	return NewRoute(r)
}

// mockProductService is a hand-rolled mock satisfying
// services.ProductServiceInterface. Each method is backed by a
// *Fn field; unset fields panic on call so missing expectations fail loudly.
type mockProductService struct {
	createFn func(ctx context.Context, e *models.Product) error
	getFn    func(ctx context.Context, id uint) (*models.Product, error)
	listFn   func(ctx context.Context, params repository.ListParams) (*repository.ListResult[*models.Product], error)
	updateFn func(ctx context.Context, e *models.Product) error
	deleteFn func(ctx context.Context, id uint) error
}

func (m *mockProductService) Create(ctx context.Context, e *models.Product) error {
	if m.createFn == nil {
		panic("createFn not set")
	}
	return m.createFn(ctx, e)
}
func (m *mockProductService) Get(ctx context.Context, id uint) (*models.Product, error) {
	if m.getFn == nil {
		panic("getFn not set")
	}
	return m.getFn(ctx, id)
}
func (m *mockProductService) List(ctx context.Context, params repository.ListParams) (*repository.ListResult[*models.Product], error) {
	if m.listFn == nil {
		panic("listFn not set")
	}
	return m.listFn(ctx, params)
}
func (m *mockProductService) Update(ctx context.Context, e *models.Product) error {
	if m.updateFn == nil {
		panic("updateFn not set")
	}
	return m.updateFn(ctx, e)
}
func (m *mockProductService) Delete(ctx context.Context, id uint) error {
	if m.deleteFn == nil {
		panic("deleteFn not set")
	}
	return m.deleteFn(ctx, id)
}

// serve registers h on a std framework and serves a single request.
func serve(method string, body io.Reader, h core.HandlerFunc) *httptest.ResponseRecorder {
	fw := std.New()
	fw.Handle(method, "/products", h)
	w := httptest.NewRecorder()
	fw.Handler().ServeHTTP(w, httptest.NewRequest(method, "/products", body))
	return w
}

func TestRoute_GET_List(t *testing.T) {
	svc := &mockProductService{
		listFn: func(ctx context.Context, params repository.ListParams) (*repository.ListResult[*models.Product], error) {
			return &repository.ListResult[*models.Product]{Page: params.Page, PerPage: params.PerPage}, nil
		},
	}
	w := serve("GET", nil, newTestRoute(svc).GET)
	if w.Code != 200 {
		t.Fatalf("status: got %d want 200; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_POST_BadJSON(t *testing.T) {
	w := serve("POST", strings.NewReader("not-json"), newTestRoute(&mockProductService{}).POST)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for bad JSON; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_PUT_MissingID(t *testing.T) {
	w := serve("PUT", strings.NewReader(`{}`), newTestRoute(&mockProductService{}).PUT)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for missing id; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_PATCH_MissingID(t *testing.T) {
	w := serve("PATCH", strings.NewReader(`{}`), newTestRoute(&mockProductService{}).PATCH)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for missing id; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_DELETE_MissingID(t *testing.T) {
	w := serve("DELETE", nil, newTestRoute(&mockProductService{}).DELETE)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for missing id; body=%q", w.Code, w.Body.String())
	}
}
