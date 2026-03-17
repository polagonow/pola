// Package textencoding provides TextEncoder and TextDecoder for the
// modernc.org/quickjs VM.
package textencoding

import (
	"fmt"

	mquickjs "modernc.org/quickjs"
)

const src = `
(function() {
	function encodeUTF8(str) {
		var result = [];
		for (var i = 0; i < str.length; ) {
			var cp = str.codePointAt(i);
			if (cp < 0x80) {
				result.push(cp);
				i++;
			} else if (cp < 0x800) {
				result.push(0xC0 | (cp >> 6), 0x80 | (cp & 0x3F));
				i++;
			} else if (cp < 0x10000) {
				result.push(0xE0 | (cp >> 12), 0x80 | ((cp >> 6) & 0x3F), 0x80 | (cp & 0x3F));
				i++;
			} else {
				result.push(
					0xF0 | (cp >> 18),
					0x80 | ((cp >> 12) & 0x3F),
					0x80 | ((cp >> 6) & 0x3F),
					0x80 | (cp & 0x3F)
				);
				i += 2; // surrogate pair in JS string
			}
		}
		return new Uint8Array(result);
	}

	function TextEncoder() {}
	TextEncoder.prototype.encoding = 'utf-8';
	TextEncoder.prototype.encode = function(str) {
		return encodeUTF8(str == null ? '' : String(str));
	};
	TextEncoder.prototype.encodeInto = function(str, dest) {
		var bytes = encodeUTF8(str == null ? '' : String(str));
		var written = Math.min(bytes.length, dest.length);
		for (var i = 0; i < written; i++) dest[i] = bytes[i];
		return { read: str.length, written: written };
	};

	function TextDecoder(enc) {
		this.encoding = enc || 'utf-8';
	}
	TextDecoder.prototype.decode = function(arr) {
		if (!arr) return '';
		if (arr.buffer) arr = new Uint8Array(arr.buffer, arr.byteOffset, arr.byteLength);
		else if (!(arr instanceof Uint8Array)) arr = new Uint8Array(arr);
		var result = [];
		var i = 0;
		while (i < arr.length) {
			var b = arr[i];
			if (b < 0x80) {
				result.push(String.fromCodePoint(b));
				i++;
			} else if ((b & 0xE0) === 0xC0) {
				result.push(String.fromCodePoint(((b & 0x1F) << 6) | (arr[i+1] & 0x3F)));
				i += 2;
			} else if ((b & 0xF0) === 0xE0) {
				result.push(String.fromCodePoint(
					((b & 0x0F) << 12) | ((arr[i+1] & 0x3F) << 6) | (arr[i+2] & 0x3F)
				));
				i += 3;
			} else {
				var cp = ((b & 0x07) << 18) | ((arr[i+1] & 0x3F) << 12) |
				         ((arr[i+2] & 0x3F) << 6) | (arr[i+3] & 0x3F);
				result.push(String.fromCodePoint(cp));
				i += 4;
			}
		}
		return result.join('');
	};

	globalThis.TextEncoder = TextEncoder;
	globalThis.TextDecoder = TextDecoder;
})();
`

// Enable installs TextEncoder and TextDecoder as globals into vm.
func Enable(vm *mquickjs.VM) error {
	if _, err := vm.Eval(src, mquickjs.EvalGlobal); err != nil {
		return fmt.Errorf("textencoding: %w", err)
	}
	return nil
}
