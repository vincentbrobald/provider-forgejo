#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Vincent Brobald
# SPDX-License-Identifier: MPL-2.0
# Vendors svalabs/terraform-provider-forgejo (MPL-2.0) into
# third_party/terraform-provider-forgejo, so that the provider's Go code is
# compiled into this Crossplane provider (no Terraform or OpenTofu binary).
#
# The only change to their code: internal/ becomes pkg/ (Go refuses imports of
# another module's internal packages) and the module path follows, with the
# import paths rewritten. Tests and examples are left out; docs/resources is
# kept for the CRD descriptions.
#
#   hack/vendor-tf-provider.sh            # the version below
# To upgrade: set VERSION and SHA256 (sha256sum of the release's source
# tarball) here and the version in internal/tfversion, run, then
# `make schema generate` and review the CRD changes (README, Building).
set -euo pipefail

VERSION=1.6.1
SHA256=36310e7011dc64b31d21ea84f78b275f9c3b96a50ec2b20d42bf66c93ace17af
URL="https://github.com/svalabs/terraform-provider-forgejo/archive/refs/tags/v${VERSION}.tar.gz"
MODULE=brobald.be/provider-forgejo/third_party/terraform-provider-forgejo

root=$(cd "$(dirname "$0")/.." && pwd)
dest="$root/third_party/terraform-provider-forgejo"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl -fsSL -o "$tmp/src.tar.gz" "$URL"
echo "$SHA256  $tmp/src.tar.gz" | sha256sum -c --quiet -
mkdir "$tmp/src"
tar xzf "$tmp/src.tar.gz" -C "$tmp/src" --strip-components=1

rm -rf "$dest"
mkdir -p "$dest"
cp "$tmp/src/LICENSE" "$tmp/src/go.mod" "$tmp/src/go.sum" "$dest/"
cp -r "$tmp/src/internal" "$dest/pkg"
# The resources' docs: descriptions and examples of the CRDs (upjet scraper).
mkdir -p "$dest/docs"
cp -r "$tmp/src/docs/resources" "$dest/docs/resources"
find "$dest/pkg" -name '*_test.go' -delete

sed -i "s|^module terraform-provider-forgejo$|module $MODULE|" "$dest/go.mod"
grep -rl 'terraform-provider-forgejo/internal/' "$dest/pkg" | xargs -r sed -i "s|\"terraform-provider-forgejo/internal/|\"$MODULE/pkg/|g"
if grep -rq '"terraform-provider-forgejo/' "$dest/pkg"; then
  echo "unexpected imports of the old module path left:" >&2
  grep -rn '"terraform-provider-forgejo/' "$dest/pkg" >&2
  exit 1
fi

cat > "$dest/VENDORED.md" <<MD
# terraform-provider-forgejo v${VERSION} (vendored)

From ${URL}
(sha256 ${SHA256}), by hack/vendor-tf-provider.sh. License: MPL-2.0 (LICENSE).

Changes: internal/ renamed to pkg/, module path set to ${MODULE}, import
paths rewritten accordingly; tests, examples and all docs but docs/resources
removed. Nothing else.
MD
echo "vendored terraform-provider-forgejo v${VERSION} into ${dest#$root/}"
