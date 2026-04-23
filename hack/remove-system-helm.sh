#!/usr/bin/env bash
# Remove any helm binary pre-installed by the runner so tests use devspace's bundled helm.
while IFS= read -r h; do
  rm -f "$h" 2>/dev/null
  [ -e "$h" ] && PATH=$(tr ':' '\n' <<<"$PATH" | grep -vxF "$(dirname "$h")" | tr '\n' ':')
done < <(type -aP helm 2>/dev/null | awk '!seen[$0]++')
export PATH; hash -r 2>/dev/null || true
