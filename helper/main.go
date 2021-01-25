package main

import (
	"github.com/loft-sh/devspace/helper/cmd"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	cmd.Execute()
}
