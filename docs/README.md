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

### Generate Config Reference (devspace.yaml)
```bash
cd ../ # main project directory
go run ./hack/docs/config/reference.go >docs/config-schema.json
```

### Generate CLI Reference
```bash
cd ../ # main project directory
go run ./hack/gen-docs.go
```

### Create Version
```bash
yarn run docusaurus docs:version 5.x
```

### Build
```
yarn build
```
This command generates static content into the `build` directory and can be served using any static contents hosting service.
