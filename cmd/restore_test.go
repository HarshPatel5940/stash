package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestWrapDecryptError_KeyMismatch(t *testing.T) {
	err := wrapDecryptError(
		errors.New("identity did not match any of the recipients: incorrect identity for recipient block"),
		"1",
		"/Users/test/.stash.key",
	)

	msg := err.Error()
	if !strings.Contains(msg, "different key") {
		t.Fatalf("expected key mismatch guidance, got: %s", msg)
	}
	if !strings.Contains(msg, "stash restore 1 -k") {
		t.Fatalf("expected command hint with -k, got: %s", msg)
	}
}
