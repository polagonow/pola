=========================================================================
currently e2e tests are coupled to testing only esbuild bundler and react render
this leaves out future bunders and renders, how can we structure our framework, or tests to make thise posible to test. 

I also want small binarys that means if user puts tags nrly taged mudules 
should be bundled in binary see (/Users/paul/projects/go-react-ssr-v2/magefile.go)
create seperate files v8go_vm.go etc in /Users/paul/projects/go-react-ssr-v2/vm/vm.go

do the same for other sections like renders
=======================================

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


SETUP magefile
https://magefile.org/zeroinstall/