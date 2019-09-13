---
title: Custom build script
id: version-v3.5.18-custom-build-script
original_id: custom-build-script
---

Instead of using Docker or kaniko, DevSpace CLI also allows you to use a custom build command. Using a custom build script allows maximum flexibility for running additional commands or using a cloud-based build environment.

```yaml
images:
  default:
    image: dscr.io/username/image
    build:
      custom:
        command: "./scripts/builder"
        args: ["--some-flag", "flag-value"]
        imageFlag: "image"
        onChange: ["./Dockerfile"]
```

The above config shows a couple of common configuration options:
- `command` either specifies a command or a path to a script.
- `args` can be used to pass arguments and flags to this custom build command or script.
- `imageFlag` is the name of the flag that DevSpace CLI will use to pass the image name including the generated tag to the build command. If `imageFlag` is not defined, DevSpace CLI will pass the image name as argument to the build command.
- `onChange` defines when DevSpace CLI should rebuild the image. If any of the files specified under `onChange` has been modified since the last build, DevSpace CLI will run the custom build command. If non of the files have changed, the build will be skipped. This behavior is automtically enabled for the correct paths when using Docker or kaniko.
