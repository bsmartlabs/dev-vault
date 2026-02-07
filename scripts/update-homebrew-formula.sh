#!/usr/bin/env bash
set -euo pipefail

# Update the Homebrew tap formula for dev-vault based on a GitHub release.
#
# Requirements:
# - gh must be authenticated OR you pass a token via HOMEBREW_TAP_GITHUB_TOKEN.
# - HOMEBREW_TAP_GITHUB_TOKEN must have write access to the tap repo.
# - The release must already exist and include checksums.txt (GoReleaser default).
#
# Usage:
#   scripts/update-homebrew-formula.sh \
#     --repo bsmartlabs/dev-vault \
#     --tag v1.2.3 \
#     --tap bsmartlabs/homebrew-dev-tools \
#     --token "$HOMEBREW_TAP_GITHUB_TOKEN"

repo=""
tag=""
tap=""
token=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo) repo="$2"; shift 2 ;;
    --tag) tag="$2"; shift 2 ;;
    --tap) tap="$2"; shift 2 ;;
    --token) token="$2"; shift 2 ;;
    -h|--help)
      sed -n '1,120p' "$0"
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

if [[ -z "$repo" || -z "$tag" || -z "$tap" || -z "$token" ]]; then
  echo "missing required args: --repo, --tag, --tap, --token" >&2
  exit 2
fi

if [[ "$tag" != v* ]]; then
  echo "tag must start with v (got: $tag)" >&2
  exit 2
fi

version="${tag#v}"

tmp="$(mktemp -d)"
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT

export GH_TOKEN="$token"

echo "Downloading checksums.txt for ${repo}@${tag}..."
gh release download "$tag" --repo "$repo" --pattern "checksums.txt" --dir "$tmp"

checksums_file="$tmp/checksums.txt"
if [[ ! -f "$checksums_file" ]]; then
  echo "checksums.txt not found in release assets" >&2
  exit 1
fi

asset_amd64="dev-vault_${version}_darwin_amd64.tar.gz"
asset_arm64="dev-vault_${version}_darwin_arm64.tar.gz"

sha_amd64="$(awk -v f="$asset_amd64" '$2==f {print $1}' "$checksums_file" | head -n 1)"
sha_arm64="$(awk -v f="$asset_arm64" '$2==f {print $1}' "$checksums_file" | head -n 1)"

if [[ -z "$sha_amd64" || -z "$sha_arm64" ]]; then
  echo "missing sha256 in checksums.txt for: $asset_amd64 and/or $asset_arm64" >&2
  echo "checksums.txt contains:" >&2
  head -n 50 "$checksums_file" >&2
  exit 1
fi

url_base="https://github.com/${repo}/releases/download/${tag}"
url_amd64="${url_base}/${asset_amd64}"
url_arm64="${url_base}/${asset_arm64}"

echo "Cloning tap repo ${tap}..."
git -c init.defaultBranch=main clone "https://x-access-token:${token}@github.com/${tap}.git" "$tmp/tap"

formula_dir="$tmp/tap/Formula"
mkdir -p "$formula_dir"

formula_path="$formula_dir/dev-vault.rb"

cat >"$formula_path" <<EOF
# typed: false
# frozen_string_literal: true

class DevVault < Formula
  desc "Scaleway Secret Manager CLI to sync -dev secrets to disk for local development"
  homepage "https://github.com/bsmartlabs/dev-vault"
  version "${version}"

  on_macos do
    if Hardware::CPU.arm?
      url "${url_arm64}"
      sha256 "${sha_arm64}"
    else
      url "${url_amd64}"
      sha256 "${sha_amd64}"
    end
  end

  def install
    bin.install "dev-vault"
  end

  test do
    system "#{bin}/dev-vault", "version"
  end
end
EOF

pushd "$tmp/tap" >/dev/null
# git diff ignores untracked files; use status so a newly created formula is detected.
if [ -z "$(git status --porcelain)" ]; then
  echo "No formula changes."
  exit 0
fi

git config user.name "bsmartbot"
git config user.email "bsmartbot@users.noreply.github.com"

git add "Formula/dev-vault.rb"
git commit -m "dev-vault ${tag}"
git push origin HEAD:main
popd >/dev/null

echo "Updated ${tap}:Formula/dev-vault.rb for ${tag}"
