package main

import (
	"fmt"
	"log"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	/*

		"github.com/devspace-cloud/devspace/cmd"
		"github.com/devspace-cloud/devspace/pkg/devspace/upgrade"*/)

var version string

func main() {
	// Get kube context to use
	//config, err := kubectl.GetClientConfigFromKubectl()
	//if err != nil {
	//	log.Fatal(err)
	//}

	client, err := kubectl.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// Get secret
	secret, err := client.Core().Secrets("default").Get("default-token-jnpf5", metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(secret.Data["token"]))
	fmt.Println(string(secret.Data["ca.crt"]))

	/*
		upgrade.SetVersion(version)

		cmd.Execute()
		os.Exit(0)*/
}
