package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

func Sleep(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: sleep seconds")
	}

	duration, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("usage: sleep seconds")
	}

	select {
	case <-ctx.Done():
	case <-time.After(time.Duration(duration) * time.Second):
	}
	return nil
}
