// Globals: MessageChannel
// Requires: queueMicrotask (01_microtask.js)
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
