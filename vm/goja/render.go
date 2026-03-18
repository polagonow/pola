package goja

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	gojalib "github.com/dop251/goja"

	"gojsx/framework"
	"gojsx/framework/globals"
)

// RenderSession holds the JS callables and stream value for a single render.
// All fields are only valid inside a vm.run callback.
type RenderSession struct {
	PullStreamFn gojalib.Callable
	DecoderObj   *gojalib.Object
	DecodeFn     gojalib.Callable
	Stream       gojalib.Value
}

// StartRender looks up __render__ and __pullStream__, instantiates a TextDecoder,
// and calls __render__ to obtain the RSC Flight ReadableStream.
func StartRender(vm *VM, exportName, propsJSON string) (*RenderSession, error) {
	var sess RenderSession
	err := vm.run(func(rt *gojalib.Runtime) error {
		renderFn, ok := gojalib.AssertFunction(rt.Get(globals.RenderFn))
		if !ok {
			return fmt.Errorf("%s is not a function", globals.RenderFn)
		}
		sess.PullStreamFn, ok = gojalib.AssertFunction(rt.Get(globals.PullStreamFn))
		if !ok {
			return fmt.Errorf("%s is not a function", globals.PullStreamFn)
		}

		decoderVal, err := rt.RunString("new TextDecoder()")
		if err != nil {
			return fmt.Errorf("TextDecoder instantiation failed: %w", err)
		}
		sess.DecoderObj = decoderVal.ToObject(rt)
		sess.DecodeFn, ok = gojalib.AssertFunction(sess.DecoderObj.Get("decode"))
		if !ok {
			return fmt.Errorf("TextDecoder.decode is not a function")
		}

		sess.Stream, err = renderFn(gojalib.Undefined(), rt.ToValue(exportName), rt.ToValue(propsJSON))
		return err
	})
	return &sess, err
}

// DrainStream polls sess until the stream is done, writing decoded chunks to w.
func DrainStream(vm *VM, w framework.StreamWriter, sess *RenderSession) (wroteAny bool, err error) {
	for {
		var done, noChunks bool
		if err := vm.run(func(rt *gojalib.Runtime) error {
			r, err := sess.PullStreamFn(gojalib.Undefined(), sess.Stream)
			if err != nil {
				return err
			}
			rObj := r.ToObject(rt)
			chunksArr := rObj.Get("chunks").ToObject(rt)
			chunksLen := int(chunksArr.Get("length").ToInteger())
			if chunksLen > 0 {
				var sb strings.Builder
				for i := range chunksLen {
					decoded, err := sess.DecodeFn(sess.DecoderObj, chunksArr.Get(strconv.Itoa(i)))
					if err != nil {
						return err
					}
					sb.WriteString(decoded.String())
				}
				w.WriteRaw([]byte(sb.String())) //nolint:errcheck
				w.Flush()
				wroteAny = true
			} else {
				noChunks = true
			}
			done = rObj.Get("done").ToBoolean()
			return nil
		}); err != nil {
			return wroteAny, err
		}
		if done {
			break
		}
		if noChunks {
			time.Sleep(5 * time.Millisecond)
		}
	}
	return wroteAny, nil
}
