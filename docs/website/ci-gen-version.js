const fs = require('fs');
const path = require('path');
const versionsFile = './versions.json';
const versions = require(versionsFile);

let latestVersion = versions[0];
let latestSidebarVersion = versions[0];

for (let i = 0; i < versions.length; i++) { 
    if (fs.existsSync('./versioned_sidebars/version-'+versions[i]+'-sidebars.json')) {
        latestSidebarVersion = versions[i];
        break; 
    }
}
const sidebarFile = `./versioned_sidebars/version-${latestVersion}-sidebars.json`;

if (latestSidebarVersion != latestVersion) {
    fs.copyFileSync(`./versioned_sidebars/version-${latestSidebarVersion}-sidebars.json`, sidebarFile);
}

const sidebarContent = fs.readFileSync(sidebarFile, 'utf8').replace(new RegExp(latestSidebarVersion, "g"), latestVersion);

fs.writeFileSync(sidebarFile, sidebarContent);

const sidebarStructure = JSON.parse(sidebarContent)[`version-${latestVersion}-docs`];

for (let sidebarGroupName in sidebarStructure) {
    const sidebarGroup = sidebarStructure[sidebarGroupName];

    for (let i = 0; i < sidebarGroup.length; i++) {
        let sidebarGroupItem = sidebarGroup[i];

        if (typeof sidebarGroupItem === "string") {
            sidebarGroupItem = {
                "ids": [sidebarGroupItem],
            };
        }

        for (let ii = 0; ii < sidebarGroupItem.ids.length; ii++) {
            const sidebarLink = sidebarGroupItem.ids[ii].replace(`version-${latestVersion}-`, "");

            // find latest version that has this page
            for (let iii = 0; iii < versions.length; iii++) {
                const version = versions[iii];
                const pagePath = `./versioned_docs/version-${version}/${sidebarLink}.md`;

                if (fs.existsSync(pagePath)) {
                    if (version != latestVersion) {
                        const pageContent = fs.readFileSync(pagePath, 'utf8').replace(new RegExp(`(id: version-)${version}`, "g"), `$1${latestVersion}`);
                        const targetPagePath = `./versioned_docs/version-${latestVersion}/${sidebarLink}.md`;

                        try {
                            fs.mkdirSync(path.dirname(targetPagePath), { recursive: true});
                        } catch(e) {}
                        fs.writeFileSync(targetPagePath, pageContent);
                    }
                    break;
                }
            }
        }
    }
}

let versionsToSave = [];
if (process.argv.length > 2 && process.argv[2]) {
    versionsToSave.push(process.argv[2]);
}
versionsToSave.push(latestVersion);

fs.writeFileSync(versionsFile, JSON.stringify(versionsToSave));
console.log(latestVersion)
