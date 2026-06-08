package nativersc

import (
	"bytes"
	"context"
	"strings"
	"testing"

	gojaengine "github.com/polagonow/pola/engine/goja"
)

// testReactShim is a minimal React.createElement + a client-reference marker and
// a page, sufficient to exercise the reconciler without the full bundler/React.
const testReactShim = `
var React = {
  createElement: function (type, props) {
    var children = Array.prototype.slice.call(arguments, 2);
    props = props || {};
    if (children.length === 1) { props = Object.assign({}, props, { children: children[0] }); }
    else if (children.length > 1) { props = Object.assign({}, props, { children: children }); }
    var key = props.key != null ? props.key : null;
    return { $$typeof: Symbol.for("react.transitional.element"), type: type, key: key, props: props, ref: null };
  }
};
var Counter = { $$typeof: Symbol.for("react.client.reference"), $$id: "components/Counter#Counter", $$async: false };
function Page(props) {
  return React.createElement("main", null, React.createElement(Counter, { initial: 3 }));
}
var __pages__ = { Index: Page };
`

// renderWith builds a goja runtime from the given bundle source, runs the
// reconciler for exportName, and returns the Flight bytes.
func renderWith(t *testing.T, bundle, exportName string, importURLs map[string]string) string {
	t.Helper()
	eng, err := gojaengine.New(bundle)
	if err != nil {
		t.Fatalf("compile bundle: %v", err)
	}
	rt, err := eng.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer rt.Dispose()

	lr, ok := rt.(loopRenderer)
	if !ok {
		t.Fatalf("runtime %T does not implement loopRenderer", rt)
	}

	var buf bytes.Buffer
	fw := newFlightWriter(&buf)
	rec := newReconciler(lr, fw, importURLs, nil)
	if err := rec.render(exportName, "{}"); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// TestReconcilerServerReference verifies that a 'use server' function passed as a
// prop to a client component is serialized as a "$F<hex>" reference plus a
// metadata row {id, bound}, matching the react-server-dom-esm client decoder.
func TestReconcilerServerReference(t *testing.T) {
	bundle := testReactShim + `
var addTodo = function () {};
addTodo.$$typeof = Symbol.for("react.server.reference");
addTodo.$$id = "actions/todos:addTodo";
function PageWithAction(props) {
  return React.createElement("form", null, React.createElement(Counter, { action: addTodo }));
}
__pages__.WithAction = PageWithAction;
` + walkSupportJS
	importURLs := map[string]string{"components/Counter": "/public/assets/Counter-x.js"}

	got := renderWith(t, bundle, "WithAction", importURLs)
	for _, want := range []string{
		`I["/public/assets/Counter-x.js","Counter"]`,
		`"id":"actions/todos:addTodo"`,
		`"bound":null`,
		`"action":"$F`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Flight output missing %q\n---\n%s", want, got)
		}
	}
}

// TestReconcilerHostAndClient verifies the end-to-end Go-driven walk: a host
// element containing a client component produces an import row plus a model row
// with a lazy reference, in the exact wire shape the esm client expects.
func TestReconcilerHostAndClient(t *testing.T) {
	bundle := testReactShim + walkSupportJS
	importURLs := map[string]string{"components/Counter": "/public/assets/Counter-x.js"}

	got := renderWith(t, bundle, "Index", importURLs)
	want := "1:I[\"/public/assets/Counter-x.js\",\"Counter\"]\n" +
		"0:[\"$\",\"main\",null,{\"children\":[\"$\",\"$L1\",null,{\"initial\":3}]}]\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestReconcilerNestedServerComponents verifies server components execute and
// their host output is inlined, fragments serialize as the fragment symbol, and
// a client ref is de-duplicated to a single import row.
func TestReconcilerNestedServerComponents(t *testing.T) {
	bundle := `
var React = {
  createElement: function (type, props) {
    var children = Array.prototype.slice.call(arguments, 2);
    props = props || {};
    if (children.length === 1) { props = Object.assign({}, props, { children: children[0] }); }
    else if (children.length > 1) { props = Object.assign({}, props, { children: children }); }
    var key = props.key != null ? props.key : null;
    return { $$typeof: Symbol.for("react.transitional.element"), type: type, key: key, props: props, ref: null };
  },
  Fragment: Symbol.for("react.fragment")
};
var Widget = { $$typeof: Symbol.for("react.client.reference"), $$id: "components/Widget#default", $$async: false };
function Inner(props) { return React.createElement("span", null, "hi"); }
function Page(props) {
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(Inner, null),
    React.createElement(Widget, null),
    React.createElement(Widget, null)
  );
}
var __pages__ = { Index: Page };
` + walkSupportJS
	importURLs := map[string]string{"components/Widget": "/public/assets/Widget-y.js"}

	got := renderWith(t, bundle, "Index", importURLs)
	want := "1:I[\"/public/assets/Widget-y.js\",\"default\"]\n" +
		"0:[\"$\",\"$Sreact.fragment\",null,{\"children\":[" +
		"[\"$\",\"span\",null,{\"children\":\"hi\"}]," +
		"[\"$\",\"$L1\",null,{}]," +
		"[\"$\",\"$L1\",null,{}]" +
		"]}]\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestReconcilerAsyncComponent verifies an async server component is deferred to a
// lazy ref in the shell and its resolved subtree streams as a later row.
func TestReconcilerAsyncComponent(t *testing.T) {
	bundle := testReactShim2 + `
function AsyncComp(props) { return Promise.resolve(React.createElement("p", null, "loaded")); }
function Page(props) { return React.createElement("div", null, React.createElement(AsyncComp, null)); }
var __pages__ = { Index: Page };
` + walkSupportJS

	got := renderWith(t, bundle, "Index", nil)
	want := "0:[\"$\",\"div\",null,{\"children\":\"$L1\"}]\n" +
		"1:[\"$\",\"p\",null,{\"children\":\"loaded\"}]\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestReconcilerSuspenseAsync verifies a Suspense boundary serializes with the
// fallback inline and async children as a lazy ref that streams later.
func TestReconcilerSuspenseAsync(t *testing.T) {
	bundle := testReactShim2 + `
function AsyncComp(props) { return Promise.resolve(React.createElement("span", null, "data")); }
function Page(props) {
  return React.createElement(
    React.Suspense,
    { fallback: React.createElement("p", null, "loading") },
    React.createElement(AsyncComp, null)
  );
}
var __pages__ = { Index: Page };
` + walkSupportJS

	got := renderWith(t, bundle, "Index", nil)
	want := "0:[\"$\",\"$Sreact.suspense\",null,{\"children\":\"$L1\",\"fallback\":[\"$\",\"p\",null,{\"children\":\"loading\"}]}]\n" +
		"1:[\"$\",\"span\",null,{\"children\":\"data\"}]\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestReconcilerBareClientRefProp verifies a client component passed as a plain
// prop value (e.g. an error boundary's fallback) serializes as a by-value "$<id>"
// reference (not "$L"), with its import row emitted. This is the pattern the
// framework error boundary uses on every page.
func TestReconcilerBareClientRefProp(t *testing.T) {
	bundle := testReactShim2 + `
var ErrorBoundary = { $$typeof: Symbol.for("react.client.reference"), $$id: "ErrorBoundary#default", $$async: false };
var ErrorFallback = { $$typeof: Symbol.for("react.client.reference"), $$id: "app/error#default", $$async: false };
function Page(props) {
  return React.createElement(ErrorBoundary, { fallback: ErrorFallback }, React.createElement("p", null, "content"));
}
var __pages__ = { Index: Page };
` + walkSupportJS
	importURLs := map[string]string{
		"ErrorBoundary": "/eb.js",
		"app/error":     "/err.js",
	}

	got := renderWith(t, bundle, "Index", importURLs)
	want := "1:I[\"/eb.js\",\"default\"]\n" +
		"2:I[\"/err.js\",\"default\"]\n" +
		"0:[\"$\",\"$L1\",null,{\"children\":[\"$\",\"p\",null,{\"children\":\"content\"}],\"fallback\":\"$2\"}]\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestReconcilerDateProp verifies a Date prop encodes as "$D"+ISO so the client
// reconstructs a real Date.
func TestReconcilerDateProp(t *testing.T) {
	bundle := testReactShim2 + `
function Page(props) {
  return React.createElement("time", { dateTime: new Date(Date.UTC(2024, 0, 2, 3, 4, 5)) }, "x");
}
var __pages__ = { Index: Page };
` + walkSupportJS

	got := renderWith(t, bundle, "Index", nil)
	want := "0:[\"$\",\"time\",null,{\"children\":\"x\",\"dateTime\":\"$D2024-01-02T03:04:05.000Z\"}]\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// testReactShim2 is like testReactShim but also exposes React.Suspense and no
// client component, for async/Suspense tests.
const testReactShim2 = `
var React = {
  createElement: function (type, props) {
    var children = Array.prototype.slice.call(arguments, 2);
    props = props || {};
    if (children.length === 1) { props = Object.assign({}, props, { children: children[0] }); }
    else if (children.length > 1) { props = Object.assign({}, props, { children: children }); }
    var key = props.key != null ? props.key : null;
    return { $$typeof: Symbol.for("react.transitional.element"), type: type, key: key, props: props, ref: null };
  },
  Suspense: Symbol.for("react.suspense"),
  Fragment: Symbol.for("react.fragment")
};
`
