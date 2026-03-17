LOOKING AT
========================================
/Users/admin/Projects/go-react-ssr-v2/vm/qjs
and /Users/admin/Projects/go-react-ssr-v2/vm/moderncquickjs and /Users/admin/Projects/go-react-ssr-v2/vm/v8go etc,
some polyfills are duplicates,
we could just create a folder with
there files should be templates and embed fs not to defined them on heap
./vm/polyfill/* them here each follows aur unifined interiface! thise will eliminate per vm polyfills and makes them easy to maintain
========================================

FOLLOW UP ON POLYFILLS
DOW WE REally need __microtaskQueue__?

ADD Away to run tests along side different vms, render engines, bundlers. for now all tests are react based, but we will support other render engines so create room for that, 

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


CRITICAL,
check all polyfill.
make sure they are all native! to direct javascript stubs


# Remove  errcheck, let's log the error but not to silience it!
//nolint:errcheck


CREATE .github actions
CRITICAL:
SETUP TRYVY VOLUNALBITY SCAN
https://trivy.dev/docs/latest/getting-started/


SETUP magefile
https://magefile.org/zeroinstall/

PLEASE WORK ON MAKING THE SHELL MORE DYNICMIC!
/Users/admin/Projects/go-react-ssr-v2/render/react/shell/template.go
