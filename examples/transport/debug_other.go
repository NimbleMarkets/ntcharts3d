//go:build !(js && wasm)

package main

// debugf is a no-op natively; see debug_js.go.
func debugf(string, ...any) {}
