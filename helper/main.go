package main

import (
	"math/rand"
	"time"

	"github.com/loft-sh/devspace/helper/cmd"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	cmd.Execute()
}
