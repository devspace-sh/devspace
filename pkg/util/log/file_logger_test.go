package log

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/util/runtime"
)

type capturingLogger struct {
	DiscardLogger
	errors []string
}

func snapshotRuntimeErrorHandlersForTest() []runtime.ErrorHandler {
	runtimeHandlerMutex.Lock()
	defer runtimeHandlerMutex.Unlock()

	return append([]runtime.ErrorHandler(nil), runtime.ErrorHandlers...)
}

func setRuntimeErrorHandlersForTest(handlers []runtime.ErrorHandler) {
	runtimeHandlerMutex.Lock()
	defer runtimeHandlerMutex.Unlock()

	runtime.ErrorHandlers = handlers
}

func firstRuntimeErrorHandlerForTest(t *testing.T) runtime.ErrorHandler {
	t.Helper()

	runtimeHandlerMutex.Lock()
	defer runtimeHandlerMutex.Unlock()

	if len(runtime.ErrorHandlers) == 0 {
		t.Fatal("expected at least one runtime error handler")
	}

	return runtime.ErrorHandlers[0]
}

func newCapturingLogger() *capturingLogger {
	return &capturingLogger{}
}

func (c *capturingLogger) Error(args ...interface{}) {
	c.errors = append(c.errors, fmt.Sprint(args...))
}

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

	t.Run("empty message and nil error", func(t *testing.T) {
		result := formatRuntimeError(nil, "")
		expected := "Runtime error occurred"
		if result != expected {
			t.Fatalf("expected %q, got %q", expected, result)
		}
	})
}

func TestNewRuntimeErrorHandler(t *testing.T) {
	logger := newCapturingLogger()
	handler := newRuntimeErrorHandler(logger)

	handler(context.Background(), errors.New("boom"), "loading config", "retry", true)

	if len(logger.errors) != 1 {
		t.Fatalf("expected one logged error, got %d", len(logger.errors))
	}
	expected := "Runtime error occurred: loading config: boom (retry=true)"
	if logger.errors[0] != expected {
		t.Fatalf("expected %q, got %q", expected, logger.errors[0])
	}
}

func TestOverrideRuntimeErrorHandler(t *testing.T) {
	originalHandlers := snapshotRuntimeErrorHandlersForTest()

	logsMutex.Lock()
	originalErrorLogger, hadErrorLogger := logs["errors"]
	logsMutex.Unlock()

	defer func() {
		setRuntimeErrorHandlersForTest(originalHandlers)

		logsMutex.Lock()
		defer logsMutex.Unlock()
		if hadErrorLogger {
			logs["errors"] = originalErrorLogger
		} else {
			delete(logs, "errors")
		}
	}()

	t.Run("installs active handler", func(t *testing.T) {
		logger := newCapturingLogger()

		logsMutex.Lock()
		logs["errors"] = logger
		logsMutex.Unlock()

		setRuntimeErrorHandlersForTest(nil)
		OverrideRuntimeErrorHandler(false)
		firstRuntimeErrorHandlerForTest(t)(context.Background(), errors.New("boom"), "loading config", "key", "item-1")
		if len(logger.errors) != 1 {
			t.Fatalf("expected one logged error, got %d", len(logger.errors))
		}
		expected := "Runtime error occurred: loading config: boom (key=item-1)"
		if logger.errors[0] != expected {
			t.Fatalf("expected %q, got %q", expected, logger.errors[0])
		}
	})

	t.Run("installs discard handler", func(t *testing.T) {
		logger := newCapturingLogger()

		logsMutex.Lock()
		logs["errors"] = logger
		logsMutex.Unlock()

		OverrideRuntimeErrorHandler(true)
		firstRuntimeErrorHandlerForTest(t)(context.Background(), errors.New("boom"), "loading config")
		if len(logger.errors) != 0 {
			t.Fatalf("expected discard handler to suppress logging, got %d messages", len(logger.errors))
		}
	})

	t.Run("can be reset in one test process", func(t *testing.T) {
		logger := newCapturingLogger()

		logsMutex.Lock()
		logs["errors"] = logger
		logsMutex.Unlock()

		OverrideRuntimeErrorHandler(true)
		firstRuntimeErrorHandlerForTest(t)(context.Background(), errors.New("discarded"), "first")
		OverrideRuntimeErrorHandler(false)
		firstRuntimeErrorHandlerForTest(t)(context.Background(), errors.New("boom"), "second")

		if len(logger.errors) != 1 {
			t.Fatalf("expected one logged error after reset, got %d", len(logger.errors))
		}
		expected := "Runtime error occurred: second: boom"
		if logger.errors[0] != expected {
			t.Fatalf("expected %q, got %q", expected, logger.errors[0])
		}
	})
}
