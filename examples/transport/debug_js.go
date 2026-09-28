//go:build js && wasm

package main

import "fmt"

// debugf prints diagnostics to the browser console (Go's println on wasm
// goes through wasm_exec.js to console.log). Native builds compile this
// out so the terminal stays clean.
func debugf(format string, args ...any) {
	println("[transport] " + fmt.Sprintf(format, args...))
}
