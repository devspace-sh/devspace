---
title: Add a predefined component
---

DevSpace provides some easy and ready to use predefined components, such as mysql, postgres, mongodb and others. You can list all available components with:
```bash
devspace list available-components
```

## Add a predefined component

Make sure you are at the root of your devspace project and have initialized the project with `devspace init`. Then run the following command in your project:
```bash
devspace add component mysql
```

You will be asked several questions about the component you want to add. Afterwards take a look at your `chart/values.yaml`:

```yaml


```
