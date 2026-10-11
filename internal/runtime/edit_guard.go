package runtime

import (
	"fmt"
	"strings"
)

// placeholderKeys are argument keys only the context projection writes.
// A tool call that carries one was copied from a stub, not authored by the
// model, so it must never reach the filesystem.
var placeholderKeys = []string{"_elided", "_hint", "_recall"}

// checkEditArgs refuses an edit whose arguments are not the model's real
// payload.
//
// Presence matters, not emptiness. Before this guard a missing new_str decoded
// to "" through str(nil), and str_replace happily deleted old_str. Session
// facb2e21ba0b31a3 lost 7 and 3 lines of main.js that way and reported
// "replaced 1 occurrence(s)" both times.
func checkEditArgs(tool string, args map[string]any, required ...string) error {
	for _, k := range placeholderKeys {
		if _, ok := args[k]; ok {
			return fmt.Errorf("refusing %s: argument %q is a context placeholder, not part of the call. Re-issue %s with the full %s value", tool, k, tool, strings.Join(required, " and "))
		}
	}
	for _, k := range required {
		if _, ok := args[k]; !ok {
			return fmt.Errorf("refusing %s: required argument %q is missing. Send it explicitly; use an empty string only when you mean an empty value", tool, k)
		}
	}
	return nil
}
