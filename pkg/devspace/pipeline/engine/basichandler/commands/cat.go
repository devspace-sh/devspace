package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"mvdan.cc/sh/v3/interp"
)

func Cat(ctx *interp.HandlerContext, args []string) error {
	if len(args) == 0 {
		_, err := io.Copy(ctx.Stdout, ctx.Stdin)
		if err != nil {
			return fmt.Errorf("cat: %v", err)
		}
		return nil
	}

	for _, arg := range args {
		file := filepath.Join(ctx.Dir, arg)
		err := printFile(file, ctx.Stdout)
		if err != nil {
			return fmt.Errorf("cat: %v", err)
		}
	}
	return nil
}

func printFile(filename string, stdout io.Writer) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(stdout, f)
	if err != nil {
		return err
	}
	return nil
}
