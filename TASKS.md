=========================================================================
add benchmarks to vms, rendering engine, each testable file should have a unit test
=======================================
discovery style should be not coupled to react, it can be used by other frameworks too
support for htmx,vue,svelte,react

fix left hook commit-msg

use solid implementaions of polyfills and you make sure you check if something is defined befefor polyfilling it!
fo example promise :https://www.promisejs.org/implementing/
=======================================

currently we are using magic strings for
__gojsx_stream__, __JSI_ etc, please scan all of them and we put it in one place such that they are maintainable!


PLEASE WORK ON MAKING THE HTML SHELL MORE DYNICMIC, WE WILL In future support html builder!
/Users/admin/Projects/go-react-ssr-v2/render/react/shell/template.go


CRITCAL
each file go file we have should have a _test.go files to have it's unit test, add them tests


SCAN AND SEE IF CODE FOLLOWS effective go guideline
https://go.dev/doc/effective_go


Add logger, there is no predefined logger
add sloger based slog

# Caching components

https://github.com/hashicorp/golang-lru


CRITICAL
let's make th project plugin based
we can let someone define there own plugins
for render (eg React), for vm,cache,etc and will introduce for css also to allow people use eg tailwind. plugin should be able to recieve notification if the files it handles changes for example css changes, for tailwind plugin needs to reload/rebuild

CRITICAL
ADD SUPPORT FOR CSS

# Remove  errcheck, let's log the error but not to silience it!
//nolint:errcheck


CREATE .github actions
CRITICAL:
SETUP TRYVY VOLUNALBITY SCAN
https://trivy.dev/docs/latest/getting-started/



=========================
https://svelte.dev/docs/kit/glossary#SSR
https://vuejs.org/api/ssr
https://angular.dev/guide/ssr
https://react.dev/
https://templ.guide/server-side-rendering/htmx/
https://htmgo.dev


# HTMX
https://templ.guide/server-side-rendering/htmx/
https://templ.guide/server-side-rendering/streaming/
https://github.com/jritsema/go-htmx-tailwind-example
https://github.com/a-h/templ/tree/main
https://data-star.dev/guide/getting_started#installation


# HTMGO
https://htmgo.dev/docs/core-concepts/raw-html



| Feature          | JS Frameworks | HTMX / templ | HTMGO   |
| ---------------- | ------------- | ------------ | ------- |
| Needs Node       | ✅             | ❌            | ❌       |
| Works with Goja  | ❌             | ✅ (optional) | ✅       |
| Complexity       | 🔴 High       | 🟢 Low       | 🟢 Low  |
| Performance      | 🟡 Medium     | 🟢 High      | 🟢 High |
| Control          | 🔴 Low        | 🟢 High      | 🟢 High |
| Plug-in friendly | 🟡            | 🟢           | 🟢      |


echo "# pola" >> README.md
git init
git add README.md
git commit -m "first commit"
git branch -M main
git remote add origin git@github.com:polagonow/pola.git
git push -u origin main
