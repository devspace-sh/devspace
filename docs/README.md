# Website

This website is built using [Docusaurus 2](https://v2.docusaurus.io/), a modern static website generator.

## Contributing

### Installation

```
yarn
```

### Development
```
yarn start
```
This command starts a local development server and open up a browser window. Most changes are reflected live without having to restart the server.

### Production Build
```
yarn build
```
This command generates static content into the `build` directory and can be served using any static contents hosting service.

### Create New Major Version
```bash
yarn run docusaurus docs:version 5.x
```

### Generate CLI Reference
```bash
cd ../ # main project directory
go run ./docs/hack/cli/main.go
```

### Generate Partials For Config (devspace.yaml)
```bash
cd ../ # main project directory

go run ./docs/hack/config/partials/main.go
```


### Generate Schema For Config (devspace.yaml)
```bash
cd ../ # main project directory

go run ./docs/hack/config/schemas/main.go
```

### Generate Function Docs
```bash
cd ../ # main project directory

go run ./docs/hack/functions/main.go
```
