package main

import (
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

func main() {
	fmt.Println("blabla world")

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
