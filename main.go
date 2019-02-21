package main

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"os"

	"github.com/covexo/devspace/cmd"
	"github.com/covexo/devspace/pkg/devspace/upgrade"
)

var version string

func main() {
	upgrade.SetVersion(version)

	cmd.Execute()
	os.Exit(0)

	//for i := 0; i < 255; i++ {
	//	fmt.Printf("%d: %s %s\n", i, ansi.Color("Hello", strconv.Itoa(i)), ansi.Color("Hello", strconv.Itoa(i)+"+b"))
	//}
}
