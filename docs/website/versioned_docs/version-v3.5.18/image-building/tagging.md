---
title: Define image tags
id: version-v3.5.18-tagging
original_id: tagging
---

If you have any image defined in your `devspace.yaml`, DevSpace will tag this image after building with a random string and push it to the defined registry. DevSpace will then replace the image name with the just build tag in memory in the resources that should be deployed (kubernetes manifests, helm chart values or component values).  

There are cases where you do not want DevSpace to tag your images with a random tag and rather want more control over the tagging process. This can be accomplished with the help of [predefined configuration variables](/docs/configuration/variables#predefined-variables).  

For example you want to tag an image with the current git commit hash, your `devspace.yaml` would look like this:
```yaml
images:
  default:
    image: myrepo/devspace
    # This tag value is used for tagging the image 
    tag: ${DEVSPACE_GIT_COMMIT}
```

You can also combine several variables together:

```yaml
images:
  default:
    image: myrepo/devspace
    # This tag value is used for tagging the image 
    tag: ${DEVSPACE_USERNAME}-devspace-${DEVSPACE_GIT_COMMIT}-${DEVSPACE_RANDOM}
```

which would result in a more complex tag. For a complete overview which variables are available take a look at [predefined configuration variables](/docs/configuration/variables#predefined-variables), of course you can also mix predefined variables with environment or user defined variables to allow for more complex use cases.  
