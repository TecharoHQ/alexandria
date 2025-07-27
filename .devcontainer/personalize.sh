#!/usr/bin/env bash
set -euo pipefail

# 1. Determine repo root and name
repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"
repo_name="$(basename "$repo_root")"

# 2. Read & normalize remote.origin.url, extract owner & repo
remote_url="$(git config --get remote.origin.url)"
remote_url="${remote_url%.git}"  # strip trailing .git if present

if [[ "$remote_url" =~ ^git@([^:]+):([^/]+)/(.+)$ ]]; then
  # SSH form: git@domain:owner/repo
  domain="${BASH_REMATCH[1]}"
  owner="${BASH_REMATCH[2]}"
  name_remote="${BASH_REMATCH[3]}"
elif [[ "$remote_url" =~ ^https?://([^/]+)/([^/]+)/(.+)$ ]]; then
  # HTTPS form: https://domain/owner/repo
  domain="${BASH_REMATCH[1]}"
  owner="${BASH_REMATCH[2]}"
  name_remote="${BASH_REMATCH[3]}"
else
  echo "❌ Error: Unrecognized remote URL: $remote_url" >&2
  exit 1
fi

# (Optionally) force domain if you always want github.com:
domain="github.com"
rename_arg="$domain/$owner/$name_remote"

echo "✅ Parsed remote → domain=$domain, owner=$owner, repo=$name_remote"

# 3. Update devcontainer.json
json_file=".devcontainer/devcontainer.json"
if [[ -f "$json_file" ]]; then
  tmp="${json_file}.tmp"
  jq --arg wf "/workspace/$name_remote" \
     '.workspaceFolder = $wf' \
     "$json_file" > "$tmp" && mv "$tmp" "$json_file"
  echo "✅ Set workspaceFolder to /workspace/$name_remote in $json_file"
else
  echo "ℹ️ Skipping: $json_file not found"
fi

# 4. Update docker-compose.yaml volume
dc=".devcontainer/docker-compose.yaml"
if [[ -f "$dc" ]]; then
  # match the literal "../:/workspace/<old>:cached"
  # capture the prefix "../:/workspace/" as \1 and the suffix ":cached" as \2
  sed -i.bak -E \
    "s@(\.\./:/workspace/)[^:]+(:cached)@\1$name_remote\2@g" \
    "$dc"
  rm -f "${dc}.bak"
  echo "✅ Volume mapping updated in $dc → ../:/workspace/$name_remote:cached"
else
  echo "ℹ️  Skipping: $dc not found"
fi

# 5. Run npm rename
echo "➡️ npm run setup:rename $domain/$owner/$name_remote"
npm run setup:rename "$domain/$owner/$name_remote"

# 6. Patch package.json (name, version, and remove setup:rename)
pkg="package.json"
if [[ -f "$pkg" ]]; then
  tmp="${pkg}.tmp"
  jq --arg n "@$owner/$name_remote" \
     --arg v "0.0.0" \
     '.name = $n
      | .version = $v
      | del(.scripts["setup:rename"])' \
     "$pkg" > "$tmp" && mv "$tmp" "$pkg"
  echo "✅ package.json → name=\"@$owner/$name_remote\", version=\"0.0.0\", removed scripts.setup:rename"
else
  echo "ℹ️  Skipping: $pkg not found"
fi

# 7. Delete CHANGELOG.md
if [[ -f "CHANGELOG.md" ]]; then
  rm -f CHANGELOG.md
  echo "✅ Deleted CHANGELOG.md"
else
  echo "ℹ️ No CHANGELOG.md to delete"
fi

# 8. Edit docker-bake.hcl registry paths
hcl="docker-bake.hcl"
if [[ -f "$hcl" ]]; then
  sed -i.bak -E \
    -e "s@ghcr\.io/xe/project-template:latest@ghcr.io/$owner/$name_remote@g" \
    "$hcl"
  rm -f "${hcl}.bak"
  echo "✅ Updated registry paths in $hcl to $owner/$name_remote"
else
  echo "ℹ️ Skipping: $hcl not found"
fi

# 9. Remove gomvp tool
go get -tool github.com/abenz1267/gomvp@none

# 10. Run formatting
npm run format

echo "✅ You're all set! Commit and push your repo to have CI run on it."
rm .devcontainer/personalize.sh