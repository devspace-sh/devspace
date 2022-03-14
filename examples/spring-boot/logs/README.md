# Develop Spring Boot application with DevSpace

## Prerequisite:

- [Docker](https://docs.docker.com/get-docker/)
- [DevSpace](https://devspace.sh/cli/docs/getting-started/installation)
- [JDK-16](https://adoptopenjdk.net)
- Kubernetes cluster
 
## Steps

### Get a Kubernetes cluster

For this example, we're using docker-desktop so that registry management is by default.
If you're choosing KIND then you'll have to create a cluster with a registry and use it.

### Clone this project

```sh
git clone git@github.com:loft-sh/devspace.git

cd examples/spring-boot/logs
```

### Create a new namespace in the cluster

```sh
kubectl create ns test-1
```

### Start developing application with DevSpace

Start DevSpace
```sh
devspace dev -n test-1
```

This is what output looks like

```sh
[info] Using namespace 'test-1'
[info] Using kube context 'docker-desktop'
[info] Skip building image 'java'
[info] Execute 'helm list --namespace test-1 --output json --kube-context docker-desktop'
[info] Skipping deployment app

#########################################################
[info] DevSpace UI available at: http://localhost:8090
#########################################################

[done] √ Port forwarding started on 8080:8080 (test-1/app-8d67f584f-krcvz)
[0:sync] Waiting for pods...
[0:sync] Starting sync...
[0:sync] Start syncing
[0:sync] Sync started on /Users/pratikjagrut/work/github.com/loft-sh/devspace/examples/spring-boot/logs/target <-> /target/ (Pod: test-1/app-8d67f584f-krcvz)
[0:sync] Waiting for initial sync to complete
[0:sync] Helper - Use inotify as watching method in container
[0:sync] Downstream - Initial sync completed
[0:sync] Upstream - Initial sync completed
[info] Starting log streaming
[app] Start streaming logs for test-1/app-8d67f584f-krcvz/container-0
[app]
[app] ############### Delete target/.jar files ###############
[app] removed 'target/app-1.jar'
[app]
[app]
[app] ############### Compile packages ###############
[app]
[app] [INFO] Scanning for projects...
[app] Downloading from central: https://repo.maven.apache.org/maven2/org/springframework/boot/spring-boot-starter-parent/2.5.5/spring-boot-starter-parent-2.5.5.pom
Downloaded from central: https://repo.maven.apache.org/maven2/org/springframework/boot/spring-boot-starter-parent/2.5.5/spring-boot-starter-parent-2.5.5.pom (8.6 kB at 9.8 kB/s)
.
.
.
.
.
.
[app]
[app] . ____ _ __ _ _
[app] /\\ / ___'_ __ _ _(_)_ __ __ _ \ \ \ \
[app] ( ( )\___ | '_ | '_| | '_ \/ _` | \ \ \ \
[app] \\/ ___)| |_)| | | | | || (_| | ) ) ) )
[app] ' |____| .__|_| |_|_| |_\__, | / / / /
[app] =========|_|==============|___/=/_/_/_/
[app] :: Spring Boot :: (v2.5.5)
[app]
[app] 2021-10-04 12:39:49.894 INFO 9 --- [ main] com.example.app.AppApplication : Starting AppApplication v1 using Java 16.0.2 on app-8d67f584f-krcvz with PID 9 (/target/app-1.jar started by root in /)
[app] 2021-10-04 12:39:49.895 INFO 9 --- [ main] com.example.app.AppApplication : No active profile set, falling back to default profiles: default
[app] 2021-10-04 12:39:50.380 INFO 9 --- [ main] o.s.b.w.embedded.tomcat.TomcatWebServer : Tomcat initialized with port(s): 8080 (http)
[app] 2021-10-04 12:39:50.386 INFO 9 --- [ main] o.apache.catalina.core.StandardService : Starting service [Tomcat]
[app] 2021-10-04 12:39:50.386 INFO 9 --- [ main] org.apache.catalina.core.StandardEngine : Starting Servlet engine: [Apache Tomcat/9.0.53]
[app] 2021-10-04 12:39:50.411 INFO 9 --- [ main] o.a.c.c.C.[Tomcat].[localhost].[/] : Initializing Spring embedded WebApplicationContext
[app] 2021-10-04 12:39:50.411 INFO 9 --- [ main] w.s.c.ServletWebServerApplicationContext : Root WebApplicationContext: initialization completed in 487 ms
[app] 2021-10-04 12:39:50.695 INFO 9 --- [ main] o.s.b.w.embedded.tomcat.TomcatWebServer : Tomcat started on port(s): 8080 (http) with context path ''
[app] 2021-10-04 12:39:50.701 INFO 9 --- [ main] com.example.app.AppApplication : Started AppApplication in 1.021 seconds (JVM running for 1.25)
```

**Goto browser and hit http://localhost:8080**

You'll see

```sh
Hello World
``` 

### Make changes to source code and see the devspace magic

Change line `return "Hello World";` to `return "Hello DevSpace";` `./src/main/java/com/example/app/AppApplication.java` and save.

```sh
[0:sync] Upstream - Upload File 'main/java/com/example/app/AppApplication.java'
[0:sync] Upstream - Upload 1 create change(s) (Uncompressed ~0.51 KB)
[0:sync] Upstream - Successfully processed 1 change(s)
[0:sync] Upstream - Restarting container
[app]
[app]
[app] ############### Restart container ###############
[app]
[app]
[app]
[app] ############### Delete target/.jar files ###############
[app]
[app] removed 'target/app-1.jar'
[app]
[app]
[app] ############### Compile packages ###############
[app]
[app] [INFO] Scanning for projects...
[app] [INFO]
[app] [INFO] --------------------------< com.example:app >---------------------------
[app] [INFO] Building app 1
[app] [INFO] --------------------------------[ jar ]---------------------------------
[app] [INFO]
[app] [INFO] --- maven-resources-plugin:3.2.0:resources (default-resources) @ app ---
[app] [INFO] Using 'UTF-8' encoding to copy filtered resources.
[app] [INFO] Using 'UTF-8' encoding to copy filtered properties files.
[app] [INFO] Copying 1 resource
[app] [INFO] Copying 0 resource
[app] [INFO]
[app] [INFO] --- maven-compiler-plugin:3.8.1:compile (default-compile) @ app ---
[app] [INFO] Changes detected - recompiling the module!
[app] [INFO] Compiling 1 source file to /target/classes
[app] [INFO]
[app] [INFO] --- maven-resources-plugin:3.2.0:testResources (default-testResources) @ app ---
[app] [INFO] Using 'UTF-8' encoding to copy filtered resources.
[app] [INFO] Using 'UTF-8' encoding to copy filtered properties files.
[app] [INFO] skip non existing resourceDirectory /src/test/resources
[app] [INFO]
[app] [INFO] --- maven-compiler-plugin:3.8.1:testCompile (default-testCompile) @ app ---
[app] [INFO] Changes detected - recompiling the module!
[app] [INFO]
[app] [INFO] --- maven-surefire-plugin:2.22.2:test (default-test) @ app ---
[app] [INFO]
[app] [INFO] --- maven-jar-plugin:3.2.0:jar (default-jar) @ app ---
[app] [INFO] Building jar: /target/app-1.jar
[app] [INFO]
[app] [INFO] --- spring-boot-maven-plugin:2.5.5:repackage (repackage) @ app ---
[app] [INFO] Replacing main artifact with repackaged archive
[app] [INFO] ------------------------------------------------------------------------
[app] [INFO] BUILD SUCCESS
[app] [INFO] ------------------------------------------------------------------------
[app] [INFO] Total time: 1.256 s
[app] [INFO] Finished at: 2021-10-04T17:23:12Z
[app] [INFO] ------------------------------------------------------------------------
[app]
[app] . ____ _ __ _ _
[app] /\\ / ___'_ __ _ _(_)_ __ __ _ \ \ \ \
[app] ( ( )\___ | '_ | '_| | '_ \/ _` | \ \ \ \
[app] \\/ ___)| |_)| | | | | || (_| | ) ) ) )
[app] ' |____| .__|_| |_|_| |_\__, | / / / /
[app] =========|_|==============|___/=/_/_/_/
[app] :: Spring Boot :: (v2.5.5)
[app]
[app] 2021-10-04 17:23:12.665 INFO 253 --- [ main] com.example.app.AppApplication : Starting AppApplication v1 using Java 16.0.2 on app-6c9595dc9b-4tz62 with PID 253 (/target/app-1.jar started by root in /)
[app] 2021-10-04 17:23:12.666 INFO 253 --- [ main] com.example.app.AppApplication : No active profile set, falling back to default profiles: default
[app] 2021-10-04 17:23:13.138 INFO 253 --- [ main] o.s.b.w.embedded.tomcat.TomcatWebServer : Tomcat initialized with port(s): 8080 (http)
[app] 2021-10-04 17:23:13.144 INFO 253 --- [ main] o.apache.catalina.core.StandardService : Starting service [Tomcat]
[app] 2021-10-04 17:23:13.144 INFO 253 --- [ main] org.apache.catalina.core.StandardEngine : Starting Servlet engine: [Apache Tomcat/9.0.53]
[app] 2021-10-04 17:23:13.175 INFO 253 --- [ main] o.a.c.c.C.[Tomcat].[localhost].[/] : Initializing Spring embedded WebApplicationContext
[app] 2021-10-04 17:23:13.176 INFO 253 --- [ main] w.s.c.ServletWebServerApplicationContext : Root WebApplicationContext: initialization completed in 476 ms
[app] 2021-10-04 17:23:13.454 INFO 253 --- [ main] o.s.b.w.embedded.tomcat.TomcatWebServer : Tomcat started on port(s): 8080 (http) with context path ''
[app] 2021-10-04 17:23:13.460 INFO 253 --- [ main] com.example.app.AppApplication : Started AppApplication in 1.009 seconds (JVM running for 1.223)
[app] 2021-10-04 17:23:16.676 INFO 253 --- [nio-8080-exec-1] o.a.c.c.C.[Tomcat].[localhost].[/] : Initializing Spring DispatcherServlet 'dispatcherServlet'
[app] 2021-10-04 17:23:16.676 INFO 253 --- [nio-8080-exec-1] o.s.web.servlet.DispatcherServlet : Initializing Servlet 'dispatcherServlet'
[app] 2021-10-04 17:23:16.676 INFO 253 --- [nio-8080-exec-1] o.s.web.servlet.DispatcherServlet : Completed initialization in 0 ms
```

And now reload the browser.

You'll see

```sh
Hello DevSpace
```

## Important Notes:

To achieve hot reloading with Java which is compiled language we need to do some important changes in the devspace configuration.

### devspace.yaml

We used the `injectRestartHelper` option available in the images and also pass the custom script path.
To know more about it please visit this [inject-restart-helper](https://devspace.sh/cli/docs/configuration/images/inject-restart-helper).

```yaml
images:
 java:
 image: ${IMAGE}
 injectRestartHelper: true
 restartHelperPath: ./restart-helper.sh
```

We're only syncing source files. To know more about syncing visit [file-synchronization](https://devspace.sh/cli/docs/configuration/development/file-synchronization).

```yaml
sync:
 - imageSelector: ${IMAGE}
 excludePaths:
 - .git/
 containerPath: /src
 localSubPath: ./src
 onUpload:
 restartContainer: true
```

#### Compile our code on each sync

To compile our code on each sync we need to do two things.

##### First is enable `injectRestartHelper` in images.

On enabling the `injectRestartHelper` option in the images section, a process wrapper script to simulate a container restart is injected by devspace during the build process.

We modified the default script a little bit and specified the path of the custom script

```yaml
injectRestartHelper: true
 restartHelperPath: ./restart-helper.sh
```

##### Second thing is restarting the container on each upload in sync

```yaml
onUpload:
 restartContainer: true
```
