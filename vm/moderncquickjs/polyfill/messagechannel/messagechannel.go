// Package messagechannel provides MessageChannel for the modernc.org/quickjs VM.
//
// Requires the microtask package to have been enabled first (reads
// queueMicrotask from the global scope at call time).
package messagechannel

import (
	"fmt"

	mquickjs "modernc.org/quickjs"
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

// Enable installs MessageChannel as a global constructor into vm.
// microtask.Enable must be called first.
func Enable(vm *mquickjs.VM) error {
	if _, err := vm.Eval(src, mquickjs.EvalGlobal); err != nil {
		return fmt.Errorf("messagechannel: %w", err)
	}
	return nil
}
