package core

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// mapSrc adapts a static map to the bindStruct value-source signature.
func mapSrc(m map[string][]string) func(string) ([]string, bool) {
	return func(name string) ([]string, bool) {
		vs, ok := m[name]
		return vs, ok
	}
}

// fakeCtx is a minimal core.Context for exercising the exported binders: Param
// and Request are real (the ShouldBind* helpers use nothing else) and JSON
// records what a must-bind would write. The embedded nil interface satisfies
// the remaining methods, which are never called.
type fakeCtx struct {
	Context
	r          *http.Request
	jsonStatus int
	jsonBody   any
}

func (f *fakeCtx) Param(name string) string { return Param(f.r, name) }
func (f *fakeCtx) Request() *http.Request   { return f.r }
func (f *fakeCtx) JSON(status int, v any) error {
	f.jsonStatus, f.jsonBody = status, v
	return nil
}

func TestBindStruct_Scalars(t *testing.T) {
	type S struct {
		Str   string  `q:"str"`
		Bool  bool    `q:"bool"`
		Int   int     `q:"int"`
		Int8  int8    `q:"int8"`
		Int64 int64   `q:"int64"`
		Uint  uint    `q:"uint"`
		U32   uint32  `q:"u32"`
		F32   float32 `q:"f32"`
		F64   float64 `q:"f64"`
	}
	m := map[string][]string{
		"str": {"hello"}, "bool": {"true"}, "int": {"-5"}, "int8": {"127"},
		"int64": {"9000000000"}, "uint": {"5"}, "u32": {"42"},
		"f32": {"1.5"}, "f64": {"2.25"},
	}
	var s S
	if err := bindStruct(&s, "q", mapSrc(m)); err != nil {
		t.Fatalf("bindStruct: %v", err)
	}
	want := S{Str: "hello", Bool: true, Int: -5, Int8: 127, Int64: 9000000000, Uint: 5, U32: 42, F32: 1.5, F64: 2.25}
	if s != want {
		t.Errorf("got %+v, want %+v", s, want)
	}
}

func TestBindStruct_Pointers(t *testing.T) {
	type S struct {
		N    *int    `q:"n"`
		Name *string `q:"name"`
		Miss *int    `q:"miss"`
	}
	var s S
	if err := bindStruct(&s, "q", mapSrc(map[string][]string{"n": {"7"}, "name": {"x"}})); err != nil {
		t.Fatalf("bindStruct: %v", err)
	}
	if s.N == nil || *s.N != 7 {
		t.Errorf("N = %v, want *7", s.N)
	}
	if s.Name == nil || *s.Name != "x" {
		t.Errorf("Name = %v, want *\"x\"", s.Name)
	}
	if s.Miss != nil {
		t.Errorf("Miss = %v, want nil (absent)", s.Miss)
	}
}

func TestBindStruct_Slices(t *testing.T) {
	type S struct {
		Tags []string `q:"tag"`
		Nums []int    `q:"num"`
	}
	var s S
	if err := bindStruct(&s, "q", mapSrc(map[string][]string{"tag": {"a", "b"}, "num": {"1", "2", "3"}})); err != nil {
		t.Fatalf("bindStruct: %v", err)
	}
	if !reflect.DeepEqual(s.Tags, []string{"a", "b"}) {
		t.Errorf("Tags = %v, want [a b]", s.Tags)
	}
	if !reflect.DeepEqual(s.Nums, []int{1, 2, 3}) {
		t.Errorf("Nums = %v, want [1 2 3]", s.Nums)
	}
}

func TestBindStruct_MissingLeavesValue(t *testing.T) {
	type S struct {
		A int `q:"a"`
		B int `q:"b"`
	}
	s := S{A: 99, B: 99}
	if err := bindStruct(&s, "q", mapSrc(map[string][]string{"a": {"5"}})); err != nil {
		t.Fatalf("bindStruct: %v", err)
	}
	if s.A != 5 {
		t.Errorf("A = %d, want 5", s.A)
	}
	if s.B != 99 {
		t.Errorf("B = %d, want untouched 99 (absent)", s.B)
	}
}

func TestBindStruct_EmptyScalarSkipped(t *testing.T) {
	type S struct {
		Name string `q:"name"`
		N    *int   `q:"n"`
	}
	s := S{Name: "keep"}
	if err := bindStruct(&s, "q", mapSrc(map[string][]string{"name": {""}, "n": {""}})); err != nil {
		t.Fatalf("bindStruct: %v", err)
	}
	if s.Name != "keep" {
		t.Errorf("Name = %q, want kept \"keep\"", s.Name)
	}
	if s.N != nil {
		t.Errorf("N = %v, want nil (empty treated as absent)", s.N)
	}
}

func TestBindStruct_ConversionError(t *testing.T) {
	type S struct {
		ID int `q:"id"`
	}
	var s S
	err := bindStruct(&s, "q", mapSrc(map[string][]string{"id": {"abc"}}))
	if err == nil || err.Error() != "invalid id" {
		t.Fatalf("err = %v, want \"invalid id\"", err)
	}
}

func TestBindStruct_DashAndUntagged(t *testing.T) {
	type S struct {
		Skip string `q:"-"`
		Bare string // untagged: must not bind from a same-named key
	}
	var s S
	if err := bindStruct(&s, "q", mapSrc(map[string][]string{"-": {"x"}, "Bare": {"y"}})); err != nil {
		t.Fatalf("bindStruct: %v", err)
	}
	if s.Skip != "" {
		t.Errorf("Skip = %q, want empty (\"-\" skips)", s.Skip)
	}
	if s.Bare != "" {
		t.Errorf("Bare = %q, want empty (no field-name fallback)", s.Bare)
	}
}

func TestBindStruct_BadTarget(t *testing.T) {
	type S struct {
		A int `q:"a"`
	}
	cases := map[string]any{
		"non-pointer":           S{},
		"nil pointer":           (*S)(nil),
		"pointer to non-struct": new(int),
	}
	for name, v := range cases {
		if err := bindStruct(v, "q", mapSrc(nil)); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

// upperText is a TextUnmarshaler used to verify the extensibility hook.
type upperText struct{ s string }

func (u *upperText) UnmarshalText(b []byte) error {
	if string(b) == "fail" {
		return errors.New("nope")
	}
	u.s = strings.ToUpper(string(b))
	return nil
}

func TestBindStruct_TextUnmarshaler(t *testing.T) {
	type S struct {
		V upperText  `q:"v"`
		P *upperText `q:"p"`
	}
	var s S
	if err := bindStruct(&s, "q", mapSrc(map[string][]string{"v": {"hi"}, "p": {"yo"}})); err != nil {
		t.Fatalf("bindStruct: %v", err)
	}
	if s.V.s != "HI" {
		t.Errorf("V = %q, want HI", s.V.s)
	}
	if s.P == nil || s.P.s != "YO" {
		t.Errorf("P = %v, want YO", s.P)
	}

	var s2 S
	if err := bindStruct(&s2, "q", mapSrc(map[string][]string{"v": {"fail"}})); err == nil || err.Error() != "invalid v" {
		t.Fatalf("err = %v, want \"invalid v\"", err)
	}
}

func TestShouldBindUri(t *testing.T) {
	type P struct {
		ID   int    `uri:"id"`
		Name string `uri:"name"`
	}
	r := httptest.NewRequest("GET", "/x", nil)
	r = WithParams(r, map[string]any{"id": "42", "name": "alice"})
	var p P
	if err := ShouldBindUri(&fakeCtx{r: r}, &p); err != nil {
		t.Fatalf("ShouldBindUri: %v", err)
	}
	if p.ID != 42 || p.Name != "alice" {
		t.Errorf("got %+v, want {42 alice}", p)
	}
}

func TestShouldBindQuery(t *testing.T) {
	type Q struct {
		ID   int      `query:"id"`
		Tags []string `query:"tag"`
		N    *int     `query:"n"`
		Bare string
	}
	r := httptest.NewRequest("GET", "/x?id=42&tag=a&tag=b&n=7&Bare=nope", nil)
	var q Q
	if err := ShouldBindQuery(&fakeCtx{r: r}, &q); err != nil {
		t.Fatalf("ShouldBindQuery: %v", err)
	}
	if q.ID != 42 {
		t.Errorf("ID = %d, want 42", q.ID)
	}
	if !reflect.DeepEqual(q.Tags, []string{"a", "b"}) {
		t.Errorf("Tags = %v, want [a b]", q.Tags)
	}
	if q.N == nil || *q.N != 7 {
		t.Errorf("N = %v, want *7", q.N)
	}
	if q.Bare != "" {
		t.Errorf("Bare = %q, want empty (no field-name fallback)", q.Bare)
	}

	bad := httptest.NewRequest("GET", "/x?id=abc", nil)
	var q2 Q
	if err := ShouldBindQuery(&fakeCtx{r: bad}, &q2); err == nil || err.Error() != "invalid id" {
		t.Fatalf("err = %v, want \"invalid id\"", err)
	}
}

func TestShouldBindHeader(t *testing.T) {
	type H struct {
		Token  string   `header:"x-token"` // lowercase tag, canonical header
		Rate   int      `header:"Rate"`
		Scopes []string `header:"X-Scope"`
	}
	r := httptest.NewRequest("GET", "/x", nil)
	r.Header.Set("X-Token", "secret")
	r.Header.Set("Rate", "10")
	r.Header.Add("X-Scope", "read")
	r.Header.Add("X-Scope", "write")
	var h H
	if err := ShouldBindHeader(&fakeCtx{r: r}, &h); err != nil {
		t.Fatalf("ShouldBindHeader: %v", err)
	}
	if h.Token != "secret" {
		t.Errorf("Token = %q, want secret (case-insensitive match)", h.Token)
	}
	if h.Rate != 10 {
		t.Errorf("Rate = %d, want 10", h.Rate)
	}
	if !reflect.DeepEqual(h.Scopes, []string{"read", "write"}) {
		t.Errorf("Scopes = %v, want [read write]", h.Scopes)
	}
}

func TestShouldBindForm(t *testing.T) {
	type F struct {
		Email string   `form:"email"`
		Tags  []string `form:"tag"`
	}

	// urlencoded body
	r := httptest.NewRequest("POST", "/x", strings.NewReader("email=a@b.com&tag=x&tag=y"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var f F
	if err := ShouldBindForm(&fakeCtx{r: r}, &f); err != nil {
		t.Fatalf("ShouldBindForm urlencoded: %v", err)
	}
	if f.Email != "a@b.com" {
		t.Errorf("Email = %q, want a@b.com", f.Email)
	}
	if !reflect.DeepEqual(f.Tags, []string{"x", "y"}) {
		t.Errorf("Tags = %v, want [x y]", f.Tags)
	}

	// multipart body
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("email", "c@d.com")
	_ = mw.WriteField("tag", "p")
	_ = mw.WriteField("tag", "q")
	_ = mw.Close()
	r2 := httptest.NewRequest("POST", "/x", &buf)
	r2.Header.Set("Content-Type", mw.FormDataContentType())
	var f2 F
	if err := ShouldBindForm(&fakeCtx{r: r2}, &f2); err != nil {
		t.Fatalf("ShouldBindForm multipart: %v", err)
	}
	if f2.Email != "c@d.com" {
		t.Errorf("Email = %q, want c@d.com", f2.Email)
	}
	if !reflect.DeepEqual(f2.Tags, []string{"p", "q"}) {
		t.Errorf("Tags = %v, want [p q]", f2.Tags)
	}
}

func TestMustBind(t *testing.T) {
	c := &fakeCtx{r: httptest.NewRequest("GET", "/x", nil)}

	// nil error: no response written, returns nil.
	if err := MustBind(c, nil); err != nil || c.jsonStatus != 0 {
		t.Fatalf("MustBind(nil): status=%d err=%v, want no write / nil", c.jsonStatus, err)
	}

	// non-nil error: writes 400 {"error": ...} and returns the same error.
	sentinel := errors.New("invalid id")
	if err := MustBind(c, sentinel); err != sentinel || c.jsonStatus != http.StatusBadRequest {
		t.Fatalf("MustBind(err): status=%d err=%v, want 400 and same err", c.jsonStatus, err)
	}
	if m, ok := c.jsonBody.(M); !ok || m["error"] != "invalid id" {
		t.Errorf("body = %#v, want M{\"error\": \"invalid id\"}", c.jsonBody)
	}
}
