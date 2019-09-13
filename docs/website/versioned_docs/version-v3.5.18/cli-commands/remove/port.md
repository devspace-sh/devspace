---
title: devspace remove port
id: version-v3.5.18-port
original_id: port
---

```bash
#######################################################
############### devspace remove port ##################
#######################################################
Removes port mappings from the devspace configuration:
devspace remove port 8080,3000
devspace remove port --label-selector=release=test
devspace remove port --all
#######################################################

Usage:
  devspace remove port [flags]

Flags:
      --all                     Remove all configured ports
  -h, --help                    help for port
      --label-selector string   Comma separated key=value selector list (e.g. release=test)
```
