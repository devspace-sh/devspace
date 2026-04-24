package kubectl

import (
	"testing"

	"k8s.io/kubectl/pkg/util/term"
)

type fakeTerminalSizeQueue struct {
	sizes []*term.TerminalSize
}

func (f *fakeTerminalSizeQueue) Next() *term.TerminalSize {
	if len(f.sizes) == 0 {
		return nil
	}

	size := f.sizes[0]
	f.sizes = f.sizes[1:]
	return size
}

func TestTermSizeQueueAdapterNext(t *testing.T) {
	t.Run("nil adapter", func(t *testing.T) {
		var adapter *termSizeQueueAdapter
		if size := adapter.Next(); size != nil {
			t.Fatalf("expected nil size, got %#v", size)
		}
	})

	t.Run("nil queue", func(t *testing.T) {
		adapter := &termSizeQueueAdapter{}
		if size := adapter.Next(); size != nil {
			t.Fatalf("expected nil size, got %#v", size)
		}
	})

	t.Run("convert terminal size", func(t *testing.T) {
		adapter := &termSizeQueueAdapter{
			q: &fakeTerminalSizeQueue{
				sizes: []*term.TerminalSize{
					{Width: 80, Height: 24},
				},
			},
		}

		size := adapter.Next()
		if size == nil {
			t.Fatal("expected terminal size, got nil")
		}
		if size.Width != 80 || size.Height != 24 {
			t.Fatalf("expected 80x24 terminal size, got %#v", size)
		}
	})
}
