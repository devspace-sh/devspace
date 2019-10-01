# DevSpace CLI Documentation
This documentation is created with Docusaurus.

## Contributing
To contribute code,
1. Fork the project
2. Clone the DevSpace CLI project: `git clone https://github.com/[YOUR_USERNAME]/devspace && cd devspace/docs`
3. Start a DevSpace for the docs page: `devspace dev`
4. Make changes
5. Test your changes on: [http://localhost:3000/docs/introduction](http://localhost:3000/docs/introduction)
6. Commit changes
7. Push commits
8. Open pull request

Docusaurus allows you to use hot reloading when editing the docs pages, so you can now edit any docs page in ./docs and Docusaurus will recompile the markdown and reload the website automatically.

## [Contribution Guidelines](../CONTRIBUTING.md)
For general information regarding contributions see: [Contribution Guidelines](../CONTRIBUTING.md)


## Creating New Versions
```bash
cd website
npm run version v4.0.3
```

**If there is a new sidebar file in `website/versioned-sidebars/` that means the sidebar has changed and you need to append the version to the `sidebarVersions` array inside `website/core/Footer.js`.**
