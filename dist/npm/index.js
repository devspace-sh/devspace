#!/usr/bin/env node
var fs = require('fs');
var path = require('path');
var exec = require('child_process').exec;
var request = require('request');

const downloadPathTemplate = "https://github.com/devspace-cloud/devspace/releases/download/v{{version}}/devspace-{{platform}}-{{arch}}";
const ARCH_MAPPING = {
    "ia32": "386",
    "x64": "amd64",
    "arm": "arm"
};
const PLATFORM_MAPPING = {
    "darwin": "darwin",
    "linux": "linux",
    "win32": "windows",
    "freebsd": "freebsd"
};

if (!(process.platform in PLATFORM_MAPPING) || !(process.arch in ARCH_MAPPING)) {
    console.error("Installation is not supported for this platform (" + process.platform + ") or architecture (" + process.arch + ")");
    return;
}
let action;

if (process.argv && process.argv.length > 2) {
    action = process.argv[2];
} else {
    console.error("Please specify a version to publish!");
    return;
}

if (action == "noop") {
    console.log("Successfully ran noop command");
    process.exit(0);
}

const packageJsonPath = path.join(".", "package.json");
if (!fs.existsSync(packageJsonPath)) {
    console.error("Unable to find package.json");
    return;
}

const requestHeaders = {
    "User-Agent": "devspace-npm-script"
};
let packageJson = JSON.parse(fs.readFileSync(packageJsonPath));

if (action == "update-version") {
    const releasesURL = "https://github.com/devspace-cloud/devspace/releases";
    
    request({uri: releasesURL, headers: requestHeaders}, function(err, res, releasePage) {
            if (res.statusCode !== 200) {
                console.error("Error requesting URL " + releasesURL + " (Status Code: " + res.statusCode + ")");
                console.error(err);
                process.exit(1);
            }
            const latestVersion = releasePage.replace(/^.*?\/devspace-cloud\/devspace\/releases\/download\/v([^\/]*)\/devspace-.*$/s, "$1");
            
            if (releasePage != latestVersion && latestVersion) {
                packageJson.version = latestVersion;

                fs.writeFileSync(packageJsonPath, JSON.stringify(packageJson, null, 4));

                process.exit(0);
            } else {
                console.error("Unable to identify latest devspace version")
                process.exit(1);
            }
        });
    return;

}
let version = packageJson.version;
let platform = PLATFORM_MAPPING[process.platform];
let arch = ARCH_MAPPING[process.arch];
let binaryName = "devspace";
let downloadPath = downloadPathTemplate.replace("{{version}}", version).replace("{{platform}}", platform).replace("{{arch}}", arch);

if (platform == "windows") {
    downloadPath += ".exe";
    binaryName += ".exe";
}

exec("npm bin", function(err, stdout, stderr) {
    let dir =  null;
    if (err || stderr || !stdout || stdout.length === 0)  {
        let env = process.env;
        if (env && env.npm_config_prefix) {
            dir = path.join(env.npm_config_prefix, "bin");
        }
    } else {
        dir = stdout.trim();
    }

    if (dir == null) callback("Error finding binary installation directory");

    const binaryPath = path.join(dir, binaryName);

    try {
        fs.unlinkSync(binaryPath);
    } catch(ex) {
        // Ignore errors when deleting the file.
    }


    if (action == "install") {
        request({uri: downloadPath, headers: requestHeaders})
            .on('error', function() {
                console.error("Error requesting URL: " + downloadPath);
                process.exit(1);
            })
            .on('response', function(res) {
                res.pipe(fs.createWriteStream(binaryPath));
            })
            .on('finish', function() {
                exit(0);
            });
    }
});
