---
title: Access Keys
---

Access keys can be used to create, delete and access spaces in a CI/CD pipeline. To create a new access key navigate to `Settings -> Access Keys` in the DevSpace Cloud UI and click on the `Create Key` button. Follow the instructions on the screen and write down the created access key.  

In the CI/CD pipeline you can now login into your account with: 
```bash
devspace login --key=ACCESS_KEY
```

As soon as you are logged in you can manage spaces with the common commands in your pipeline like `devspace create space`, `devspace use space` and `devspace remove space`.  
