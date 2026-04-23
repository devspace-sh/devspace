package log

import (
	"errors"
	"testing"
)

func TestFormatRuntimeError(t *testing.T) {
	t.Run("error only", func(t *testing.T) {
		result := formatRuntimeError(errors.New("boom"), "")
		expected := "Runtime error occurred: boom"
		if result != expected {
			t.Fatalf("expected %q, got %q", expected, result)
		}
	})

	t.Run("message error and fields", func(t *testing.T) {
		result := formatRuntimeError(errors.New("boom"), "Loading client cert failed", "key", "item-1", "retry", true)
		expected := "Runtime error occurred: Loading client cert failed: boom (key=item-1, retry=true)"
		if result != expected {
			t.Fatalf("expected %q, got %q", expected, result)
		}
	})

	t.Run("message and fields without error", func(t *testing.T) {
		result := formatRuntimeError(nil, "Loading client cert failed", "key", "item-1")
		expected := "Runtime error occurred: Loading client cert failed (key=item-1)"
		if result != expected {
			t.Fatalf("expected %q, got %q", expected, result)
		}
	})

	t.Run("odd fields", func(t *testing.T) {
		result := formatRuntimeError(errors.New("boom"), "", "key")
		expected := "Runtime error occurred: boom (key=<missing>)"
		if result != expected {
			t.Fatalf("expected %q, got %q", expected, result)
		}
	})
}
