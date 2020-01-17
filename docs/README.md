# DevSpace CLI Documentation
This documentation is created with Docusaurus.

## Contributing
Follow these steps to contribute to the documentation:
1. Fork the project
2. Clone the DevSpace CLI project: `git clone https://github.com/[YOUR_USERNAME]/devspace`
3. Switch to this folder: `cd devspace/docs`
4. Either:
   1. Create a space: `devspace create space devspace-docs`
   2. Use an existing space: `devspace use space devspace-docs`
5. Start development mode: `devspace dev` (wait until the browser opens)
6. Make changes
7. Test your changes on: [http://localhost:3000/docs/introduction](http://localhost:3000/docs/introduction)
8. Commit changes
9.  Push commits
10. Open pull request

Docusaurus allows you to use hot reloading when editing the docs pages, so you can now edit any docs page in ./docs and Docusaurus will recompile the markdown and reload the website automatically.

## [Contribution Guidelines](../CONTRIBUTING.md)
For general information regarding contributions see: [Contribution Guidelines](../CONTRIBUTING.md)

## Creating New Versions

### 1. Generate Command Docs 
```bash
cd ../ # main project directory
go run -mod= ./hack/gen-docs.go
```

### 2. Create Version
```bash
cd website
npm run version v4.0.3
```

### 3. Update Sidebars
**If there is a new sidebar file in `website/versioned-sidebars/` that means the sidebar has changed and you need to:** 
- (if needed:) create a new CSS style for the sidebar in `website/static/css/versions/SIDEBAR_VERSION/style.css`
- APPEND the DevSpace version as key to the `sidebarVersions` objects inside `website/core/Footer.js` and define which sidebar version (value) should be used
