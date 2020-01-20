package main

import (
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello from Service 2!")
}

func main() {
	fmt.Println("Started server on :8086")

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8086", nil)
}
