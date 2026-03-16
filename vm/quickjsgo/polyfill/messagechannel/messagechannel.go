// Package messagechannel provides MessageChannel for the QuickJS VM.
//
// Requires the microtask package to have been enabled first (reads
// queueMicrotask from the global scope at call time).
package messagechannel

import (
	"fmt"

	quickjs "github.com/buke/quickjs-go"
)

const src = `
(function() {
	var portProto = {
		postMessage: function(data) {
			var partner = this._partner;
			if (!partner) return;
			queueMicrotask(function() {
				if (typeof partner.onmessage === 'function') {
					partner.onmessage({ data: data });
				}
			});
		},
		close: function() {}
	};

	function MessageChannel() {
		var port1 = Object.create(portProto);
		port1.onmessage = null;
		port1._partner = null;

		var port2 = Object.create(portProto);
		port2.onmessage = null;
		port2._partner = null;

		port1._partner = port2;
		port2._partner = port1;

		this.port1 = port1;
		this.port2 = port2;
	}

	globalThis.MessageChannel = MessageChannel;
})();
`

// Enable installs MessageChannel as a global constructor into ctx.
// microtask.Enable must be called first.
func Enable(ctx *quickjs.Context) error {
	ret := ctx.Eval(src, quickjs.EvalFileName("messagechannel.js"))
	defer ret.Free()
	if ret.IsException() {
		return fmt.Errorf("messagechannel: %w", ctx.Exception())
	}
	return nil
}
