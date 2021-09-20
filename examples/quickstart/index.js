const express = require('express')
const app = express()
const port = 3000

app.get('/', (req, res) => {
  res.send(`
    <html>
        <head>
            <link rel="stylesheet" href="https://devspace.sh/css/quickstart.css">
        </head>
        <body>
            <img src="https://devspace.sh/images/congrats.gif" />
            <h1>You deployed this project with DevSpace!</h1>
            <div>
                <h2>Now it's time to code:</h2>
                <ol>
                    <li>Edit this text in <code>index.js</code> and save the file</li>
                    <li>Check the logs to see how DevSpace restarts your container</li>
                    <li>Reload browser to see the changes</li>
                </ol>
            </div>
        </body>
    </html>
    `)
})

app.listen(port, () => console.log("Example app listening on http://localhost:" + port))