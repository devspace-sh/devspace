package main

import (
	"encoding/json"
	"github.com/invopop/jsonschema"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"k8s.io/klog/v2"
	"os"
)

func main() {
	schema := jsonschema.Reflect(&latest.Config{})
	b, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		panic(err)
	}
	f, err := os.Create("devspace-schema.json")
	if err != nil {
		klog.Fatal(err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			klog.Fatal(err)
		}
	}(f)
	_, err = f.WriteString(string(b))
	if err != nil {
		klog.Fatal(err)
	}
}
