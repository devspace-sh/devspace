package types

import "context"

const DotReplacement = "___devspace___"

type ExecHandler interface {
	ExecHandler(ctx context.Context, args []string) error
}
