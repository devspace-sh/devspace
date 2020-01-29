package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
		<html>
			<head>
				<link rel="stylesheet" href="https://devspace.cloud/quickstart.css">
			</head>
			<body>
				<img src="https://devspace.cloud/img/congrats.gif" />
				<h1>You deployed this project with DevSpace!</h1>
				<div>
					<h2>Now it's time to start the development mode:</h2>
					<ol>
						<li>Press CTRL+C or ENTER to terminate <code>devspace open</code></li>
						<li>Run: <code>devspace dev</code></li>
						<li>Edit this text in <code>main.go</code> and save the file</li>
						<li>Check the logs to see how <code>CompileDaemon</code> recompiles and restarts this project</li>
						<li>Reload browser to see the changes</li>
					</ol>
				</div>
			</body>
		</html>
		`)
	})

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(":8080", nil)
}
