package resolve

import "strings"

// Browser built-in constructors and globals: calls to them are external (a project function of the same name still wins).
func init() {
  for _, w := range strings.Fields("FormData URLSearchParams Blob FileReader XMLHttpRequest Headers Request Response Image Audio WebSocket AbortController IntersectionObserver MutationObserver ResizeObserver CustomEvent Event Notification BigInt Uint8Array Float64Array ArrayBuffer RegExp WeakMap WeakSet queueMicrotask requestAnimationFrame cancelAnimationFrame structuredClone") {
    builtins[w] = true
  }
}
