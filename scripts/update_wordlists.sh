#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="fuzz4bounty"
TMP_DIR="$(mktemp -d)"

echo "[*] Updating wordlists..."


# Use YQ_PATH if set (for GitHub Actions), otherwise fallback to 'yq' (for local/dev use)
YQ_BIN="${YQ_PATH:-yq}"

while IFS= read -r file; do
  URL=$($YQ_BIN e ".\"$file\".url" sources.yaml)

  echo "[+] Fetching $file"
  echo "# Source: $URL" > "$TMP_DIR/$file"
  curl -sSL "$URL" \
    | sed '/^\s*#/d;/^\s*$/d' \
    | sort -u >> "$TMP_DIR/$file"

  if [[ -f "$BASE_DIR/$file" ]]; then
    if ! diff -q "$BASE_DIR/$file" "$TMP_DIR/$file" >/dev/null; then
      mv "$TMP_DIR/$file" "$BASE_DIR/$file"
      echo "    -> Updated"
    else
      echo "    -> No changes"
    fi
  else
    mv "$TMP_DIR/$file" "$BASE_DIR/$file"
    echo "    -> New file added"
  fi

done < <($YQ_BIN e 'keys | .[]' sources.yaml)

rm -rf "$TMP_DIR"
