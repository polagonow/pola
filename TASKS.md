FOLLOW UP ON POLYFILLS

let polyfill tests be close to what they are testing,
for example /Users/paul/projects/go-react-ssr-v2/vm/polyfill/01_microtask.js should have /Users/paul/projects/go-react-ssr-v2/vm/polyfill/01_microtask_test.go to follow domain driven develpment

if they have any drivers, configs to share, let it it be in them root ./test folder

DOW WE REally need __microtaskQueue__?
=========================================================================




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
