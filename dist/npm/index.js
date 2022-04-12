#!/usr/bin/env node
const fs = require("fs");
const path = require("path");
const execSync = require("child_process").execSync;
const request = require("request");
const Spinner = require("cli-spinner").Spinner;
const inquirer = require('inquirer');
const findProcess = require('find-process');

const downloadPathTemplate =
  "https://github.com/loft-sh/devspace/releases/download/v{{version}}/devspace-{{platform}}-{{arch}}";
const ARCH_MAPPING = {
  ia32: "386",
  x64: "amd64",
  x86_64: "amd64",
  arm: "arm",
  arm64: "arm64",
  aarch64: "arm"
};
const PLATFORM_MAPPING = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
  freebsd: "freebsd"
};

if (
  !(process.platform in PLATFORM_MAPPING) ||
  !(process.arch in ARCH_MAPPING)
) {
  console.error(
    "Installation is not supported for this platform (" +
      process.platform +
      ") or architecture (" +
      process.arch +
      ")"
  );
  return;
}
let action;

if (process.argv && process.argv.length > 2) {
  action = process.argv[2];
} else {
  console.error("Please specify a version to publish!");
  return;
}

if (action === "noop") {
  console.log("Successfully ran noop command");
  process.exit(0);
}

const packageJsonPath = path.join(__dirname, "package.json");
if (!fs.existsSync(packageJsonPath)) {
  console.error("Unable to find package.json");
  return;
}

const requestHeaders = {
  "User-Agent": "devspace-npm-script"
};
let packageJson = JSON.parse(fs.readFileSync(packageJsonPath));

const getLatestVersion = function(callback, includePreReleases) {
  const releasesURL = "https://github.com/loft-sh/devspace/releases";

  request({ uri: releasesURL, headers: requestHeaders }, function(
    err,
    res,
    releasePage
  ) {
    if (res.statusCode !== 200) {
      console.error(
        "Error requesting URL " +
          releasesURL +
          " (Status Code: " +
          res.statusCode +
          ")"
      );
      console.error(err);
      process.exit(1);
    }
    let versionRegex = 
    /^.*?\/loft-sh\/devspace\/releases\/download\/v([^\/-]*)\/devspace-.*$/s;

    if (includePreReleases) {
      versionRegex = 
      /^.*?\/loft-sh\/devspace\/releases\/download\/v([^\/]*)\/devspace-.*$/s;
    }
    
    const latestVersion = releasePage.replace(versionRegex,
      "$1"
    );

    if (releasePage !== latestVersion && latestVersion) {
      callback(latestVersion);
    } else {
      console.error("Unable to identify latest devspace version");
      process.exit(2);
    }
  });
};

if (action === "update-version") {
  getLatestVersion(function(latestVersion) {
    packageJson.version = latestVersion;

    fs.writeFileSync(packageJsonPath, JSON.stringify(packageJson, null, 4));

    process.exit(0);
  }, true);
  return;
}

if (action === "get-tag") {
  getLatestVersion(function(latestVersion) {
    let tagRegex = /^.*-([a-z]*)(\.)?([0-9]*)?$/i
    let tag = "latest"
    
    if (latestVersion.match(tagRegex)) {
      tag = latestVersion.replace(tagRegex, "$1")
    }
    process.stdout.write(tag);
    process.exit(0);
  }, true);
  return;
}

/**
 * Remove directory recursively
 * @param {string} dir_path
 * @see https://stackoverflow.com/a/42505874/3027390
 */
function rimraf(dir_path) {
  if (fs.existsSync(dir_path)) {
      fs.readdirSync(dir_path).forEach(function(entry) {
          let entry_path = path.join(dir_path, entry);
          if (fs.lstatSync(entry_path).isDirectory()) {
              rimraf(entry_path);
          } else {
              fs.unlinkSync(entry_path);
          }
      });
      fs.rmdirSync(dir_path);
  }
}

let continueProcess = function(askRemoveGlobalFolder) {
  let version = packageJson.version;
  let platform = PLATFORM_MAPPING[process.platform];
  let arch = ARCH_MAPPING[process.arch];
  let downloadExtension = ".dl";
  let binaryName = packageJson.name;

  if (platform === PLATFORM_MAPPING.win32) {
    binaryName += ".exe";
  }

  let normalizePath = function(p) {
    let re = path.normalize(p).replace(/(\r)?\n/g, "");

    try {
      return fs.realpathSync(re);
    } catch(e) {
      return re;
    }
  }

  let packageDir = normalizePath(__dirname)
  let fallbackGlobalDir = "/usr/local/bin";
  let globalInstall = false;
  let globalDir = null;

  if (process.argv.length > 3 && fs.existsSync(normalizePath(process.argv[3]))) {
    globalDir = normalizePath(process.argv[3]);
    dotBinDir = normalizePath(path.join(globalDir, "..", "..", ".bin"));
    
    if (fs.existsSync(normalizePath(path.join(dotBinDir, "devspace")))) {
      globalDir = normalizePath(dotBinDir);
    }
  }

  try {
    let yarnGlobalDir = normalizePath(path.join(execSync('yarn global dir').toString(), "node_modules"));
    let yarnLink = normalizePath(path.join(yarnGlobalDir, packageJson.name));
    let yarnLinkExists = fs.existsSync(yarnLink) && yarnLink === packageDir;

    if (yarnLinkExists || packageDir.startsWith(yarnGlobalDir)) {
      try {
        globalDir = normalizePath(execSync('yarn global bin').toString());
        globalInstall = true;
      } catch(e) {
        console.log(e);
      }
    }
  } catch(e) {}

  try {
    let npmGlobalDir = normalizePath(execSync('npm root -g').toString());
    let npmLink = normalizePath(path.join(npmGlobalDir, packageJson.name));
    let npmLinkExists = fs.existsSync(npmLink) && npmLink === packageDir;

    if (npmLinkExists || !globalDir || packageDir.startsWith(npmGlobalDir)) {
      try {
        globalDir = normalizePath(execSync('npm bin -g').toString());
        globalInstall = true;
      } catch(e) {
        console.error(e);
      }
    }
  } catch(e) {}
  
  if (globalDir === null) {
    if (platform === PLATFORM_MAPPING.win32) {
      console.error("Error finding binary installation directory");
      process.exit(3);
    }
    globalDir = fallbackGlobalDir;
  }
  
  try {
    fs.mkdirSync(globalDir, { recursive: true });
  } catch(e) {}
  
  let binaryPath = path.join(globalDir, binaryName);
  if (process.argv.length > 3 && fs.existsSync(normalizePath(process.argv[3]))) {
    let binaryDir = normalizePath(process.argv[3]);
    let possibleBinaryPath = path.join(binaryDir, binaryName)
    if (fs.existsSync(possibleBinaryPath)) {
      binaryPath = possibleBinaryPath;
    }
  }

  try {
    fs.unlinkSync(binaryPath + downloadExtension);
  } catch (e) {}

  try {
    fs.unlinkSync(path.join(globalDir, "." + binaryName + ".old"));
  } catch (e) {}

  let removeScripts = function(allScripts) {
    if (platform === PLATFORM_MAPPING.win32) {
      if (allScripts) {
        try {
          fs.unlinkSync(binaryPath.replace(/\.exe$/i, ""));
        } catch (e) {}
      }
      
      // Remove bin/devspace.ps1 file because it can cause issues
      try {
        fs.unlinkSync(binaryPath.replace(/\.exe$/i, ".ps1"));
      } catch (e) {}
    }
  }
  
  if (action === "install") {
    removeScripts(false);

    if (platform === PLATFORM_MAPPING.win32) {
      if (globalInstall) {
        // Remove bin/devspace.cmd file because it can cause issues
        try {
          fs.unlinkSync(binaryPath.replace(/\.exe$/i, ".cmd"));
        } catch (e) {}

        // Copy #PROJECT_DIR/bin/devspace.cmd file to $NPM_GLOBAL/bin/devspace.cmd
        try {
          fs.copyFileSync(path.join(__dirname, "bin", "devspace.cmd"), binaryPath.replace(/\.exe$/i, ".cmd"));
        } catch (e) {}
      }
    }
  }
  else if (action === "uninstall") {
    try {
      fs.unlinkSync(binaryPath);
    } catch (e) {}

    try {
      fs.unlinkSync(path.join(fallbackGlobalDir, binaryName));
    } catch (e) {}

    // Remove bin/devspace.cmd
    try {
      fs.unlinkSync(binaryPath.replace(/\.exe$/i, ".cmd"));
    } catch (e) {}
    
    removeScripts(true);

    if (askRemoveGlobalFolder && process.stdout.isTTY) {
      let removeGlobalFolder = function() {
        try {
          let homedir = require('os').homedir();
          rimraf(homedir + path.sep + ".devspace");
        } catch (e) {
          console.error(e)
        }
      };

      inquirer
        .prompt([
          {
            type: "list",
            name: "checkRemoveGlobalFolder",
            message: "Do you want to remove the global DevSpace config folder ~/.devspace?",
            choices: ["no", "yes"],
          },
        ])
        .then(answers => {
          if (answers.checkRemoveGlobalFolder === "yes") {
            removeGlobalFolder();
          }
        });
    } else {
      console.warn("DevSpace will not remove the global ~/.devspace folder without asking. This uninstall call is being executed in a non-interactive environment.")
    }
  } else {
    if (action === "finish-install") {
      cleanPathVar = process.env.PATH.replace(/(^|;)[a-z]:/gi, path.delimiter).replace(/(\\)+/g, '/');
      cleanGlobalDir = globalDir.replace(/(^|;)[a-z]:/gi, '').replace(/(\\)+/g, '/').trimRight("/");
    
      if (cleanPathVar.split(path.delimiter).indexOf(cleanGlobalDir) === -1 && cleanPathVar.split(path.delimiter).indexOf(cleanGlobalDir + "/") === -1) {
        console.error("\n\n################################################\nWARNING: npm binary directory NOT in $PATH environment variable: " + globalDir + "\n################################################\n\n");
    
        if (globalInstall) {
          process.exit(4)
        }
      }

      const showRootError = function() {
        console.error("\n############################################");
        console.error(
          "Failed to download DevSpace CLI due to permission issues!\n"
        );
        console.error("There are two options to fix this:");
        console.error("1. Do not run 'npm install' as root (recommended)");
        console.error(
          "2. Run this command: npm install --unsafe-perm=true -g devspace"
        );
        console.error("   You may need to run this command using sudo.");
        console.error("############################################\n");
        process.exit(5);
      };

      const downloadRelease = function(version) {
        let downloadPath = downloadPathTemplate
          .replace("{{version}}", version)
          .replace("{{platform}}", platform)
          .replace("{{arch}}", arch);

        if (platform === PLATFORM_MAPPING.win32) {
          downloadPath += ".exe";
        }

        console.log("Download DevSpace CLI release: " + downloadPath + "\n");

        const spinner = new Spinner(
          "%s Downloading DevSpace CLI... (this may take a minute)"
        );
        spinner.setSpinnerString("|/-\\");
        spinner.start();

        let writeStream = fs
          .createWriteStream(binaryPath + downloadExtension)
          .on("error", function(err) {
            spinner.stop(true);
            console.error("Unable to write stream: " + err)
            showRootError();
          });

        request({ uri: downloadPath, headers: requestHeaders, encoding: null })
          .on("error", function() {
            spinner.stop(true);
            console.error("Error requesting URL: " + downloadPath);
            process.exit(6);
          })
          .on("response", function(res) {
            if (res.statusCode !== 200) {
              writeStream.end();
              spinner.stop(true);

              if (res.statusCode === 404) {
                console.error("Release version " + version + " not found.\n");

                getLatestVersion(function(latestVersion) {
                  if (latestVersion !== version) {
                    console.log(
                      "Downloading latest stable release instead. Latest version is: " +
                        latestVersion +
                        "\n"
                    );

                    downloadRelease(latestVersion);
                  }
                });
              } else {
                console.error(
                  "Error requesting URL " +
                    downloadPath +
                    " (Status Code: " +
                    res.statusCode +
                    ")"
                );
                console.error(err);
                process.exit(7);
              }
            } else {
              try {
                res.pipe(writeStream);
              } catch (e) {
                console.error("Unable to write stream: " + e)
                showRootError();
              }
            }
          })
          .on("end", function() {
            writeStream.end();
            spinner.stop(true);

            try {
              fs.chmodSync(binaryPath + downloadExtension, "0755");
            } catch (e) {
              console.error("Unable to chmod: " + e)
              showRootError();
            }

            try {
              fs.renameSync(binaryPath + downloadExtension, binaryPath);
            } catch (e) {
              console.log(e);
              console.error("\nRenaming release binary failed. Please copy file manually:\n from: " + binaryPath + downloadExtension + "\n to: " + binaryPath + "\n");
              process.exit(8);
            }

            removeScripts(true);
          });
      };

      downloadRelease(version);
    }
  }
}

if (process.ppid > 1) {
  findProcess('pid', process.ppid)
    .then(function (list) {
      if (list.length === 1 && list[0].ppid > 1) {
        findProcess('pid', list[0].ppid)
          .then(function (list) {
            if (list.length === 1 && /((npm-cli.js("|')\s+up(date)?)|(yarn.js("|')\s+(global\s+)?upgrade))\s+.*((\/)|(\\)|(\s))devspace((\/)|(\\)|(\s)|$)/.test(list[0].cmd)) {
              continueProcess(false);
            } else {
              continueProcess(true);
            }
          }, function () {
            continueProcess(true);
          })
      } else {
        continueProcess(true);
      }
    }, function () {
      continueProcess(true);
    })
} else {
  continueProcess(true);
}
