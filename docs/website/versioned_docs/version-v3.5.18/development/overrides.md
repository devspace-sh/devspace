---
title: Configure overrides (e.g. entrypoint)
id: version-v3.5.18-overrides
original_id: overrides
---

When developing your application, it is often useful to override the image entrypoints or use a separate Dockerfile. DevSpace applies this special configuration only during `devspace dev` and not during `devspace deploy`. 

## Configuring a different Dockerfile during `devspace dev`
You can tell DevSpace to use a different Dockerfile during `devspace dev` within the `dev.overrideImages` section of `devspace.yaml`:
```yaml
dev:
  overrideImages:
  - name: default
    dockerfile: ./development/Dockerfile.development
    # Optional use different context
    # context: ./development
images:
  default:
    image: dscr.io/my-username/my-image
```

## Configuring entrypoint overrides
You can also configure an entrypoint override, then `devspace dev` will do the following:

1. Load the Dockerfile for this image
2. Override the entrypoint for this image **in-memory**
3. Building your image with overridden entrypoint
4. Pushing the image to the registry under a randomly generated tag

> Overriding an entrypoint will **not** change your Dockerfile. Image overriding happen entirely in-memory before building the image.

You can configure entrypoint overrides within the `dev.overrideImages` section of `devspace.yaml`. 
```yaml
dev:
  overrideImages:
  - name: default
    entrypoint:
    - sleep
    - 9999999
images:
  default:
    image: dscr.io/my-username/my-image
```
The example above defines an image `default` which will be built and then pushed to `dscr.io/my-username/my-image` whenevery you run `devspace deploy` or `devspace dev`. When running `devspace dev`, however, the `dev.overrideImages` configuration would define that this image with name `default` would be build with an overridden entrypoint. 

> Everything within the `dev` section of the DevSpace config (including `overrideImages`) will  only be applied when running `devspace dev`.

In the example above, the result would be that the container that uses the `default` image would not start the application but just enter sleep mode. This has the advantage that you can start the application manually by entering a command in the terminal that is opened by `devspace dev`.

---
## FAQ

<details>
<summary>
### When should I use entrypoint overrides?
</summary>
Common use cases for overriding entrypoints are:
1. You want to start your application in dev mode with hot reloading (e.g. `npm run watch` using nodemon instead of `npm start`).
2. You want to increase the log level or set environment variables before starting your app (e.g. `NODE_ENV=development && npm start`).
3. You want to start a container without starting your application (e.g. `sleep 99999999`) because you want to start the application manually via the [terminal proxy](/docs/development/terminal).
</details>

<details>
<summary>
### Will image overrides also work for `devspace deploy`?
</summary>
**No.** Image overriding will only be executing when running `devspace dev`. It is recommended that you define a production version of your application which is supposed to be executed when running `devspace deploy`. Overrides are meant to override this production configuration when you are developing your application with `devspace dev`.
</details>
