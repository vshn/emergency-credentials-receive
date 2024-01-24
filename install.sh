#!/usr/bin/env bash

# This script is a modified version of the script from the kustomize repo:
# https://github.com/kubernetes-sigs/kustomize/blob/master/hack/install_kustomize.sh
## Copyright 2022 The Kubernetes Authors.
## SPDX-License-Identifier: Apache-2.0

# If no argument is given -> Downloads the most recently released
# emergency-credentials-receive binary to your current working directory.
# (e.g. 'install.sh')
#
# If one argument is given ->
# If that argument is in the format of #.#.#, downloads the specified
# version of the emergency-credentials-receive binary to your current working directory.
# If that argument is something else, downloads the most recently released
# emergency-credentials-receive binary to the specified directory.
# (e.g. 'install.sh 0.1.0' or 'install.sh $(go env GOPATH)/bin')
#
# If two arguments are given -> Downloads the specified version of the
# emergency-credentials-receive binary to the specified directory.
# (e.g. 'install.sh 0.1.0 $(go env GOPATH)/bin')
#
# Fails if the file already exists.

set -e

# Unset CDPATH to restore default cd behavior. An exported CDPATH can
# cause cd to output the current directory to STDOUT.
unset CDPATH

where=$PWD

release_url=https://api.github.com/repos/vshn/emergency-credentials-receive/releases
if [ -n "$1" ]; then
  if [[ "$1" =~ ^[0-9]+(\.[0-9]+){2}$ ]]; then
    version=v$1
    release_url=${release_url}/tags/$version
  elif [ -n "$2" ]; then
    echo "The first argument should be the requested version."
    exit 1
  else
    where="$1"
  fi
fi

if [ -n "$2" ]; then
  where="$2"
fi

if ! test -d "$where"; then
  echo "$where does not exist. Create it first."
  exit 1
fi

# Emulates `readlink -f` behavior, as this is not available by default on MacOS
# See: https://stackoverflow.com/questions/1055671/how-can-i-get-the-behavior-of-gnus-readlink-f-on-a-mac
function readlink_f {
  TARGET_FILE=$1

  cd "$(dirname "$TARGET_FILE")"
  TARGET_FILE=$(basename "$TARGET_FILE")

  # Iterate down a (possible) chain of symlinks
  while [ -L "$TARGET_FILE" ]
  do
      TARGET_FILE=$(readlink "$TARGET_FILE")
      cd "$(dirname "$TARGET_FILE")"
      TARGET_FILE=$(readlink "$TARGET_FILE")
  done

  # Compute the canonicalized name by finding the physical path
  # for the directory we're in and appending the target file.
  PHYS_DIR=$(pwd -P)
  RESULT=$PHYS_DIR/$TARGET_FILE
  echo "$RESULT"
}

function find_release_url() {
  local releases=$1
  local opsys=$2
  local arch=$3

  echo "${releases}" |\
    grep "browser_download.*${opsys}_${arch}" |\
    cut -d '"' -f 4 |\
    sort -V | tail -n 1
}

where="$(readlink_f "$where")/"

if [ -f "${where}emergency-credentials-receive" ]; then
  echo "${where}emergency-credentials-receive exists. Remove it first."
  exit 1
elif [ -d "${where}emergency-credentials-receive" ]; then
  echo "${where}emergency-credentials-receive exists and is a directory. Remove it first."
  exit 1
fi

tmpDir=$(mktemp -d)
if [[ ! "$tmpDir" || ! -d "$tmpDir" ]]; then
  echo "Could not create temp dir."
  exit 1
fi

function cleanup {
  rm -rf "$tmpDir"
}

trap cleanup EXIT ERR

pushd "$tmpDir" >& /dev/null

opsys=windows
if [[ "$OSTYPE" == linux* ]]; then
  opsys=linux
elif [[ "$OSTYPE" == darwin* ]]; then
  opsys=darwin
fi

# Supported values of 'arch': amd64, arm64, ppc64le, s390x
case $(uname -m) in
x86_64)
    arch=amd64
    ;;
arm64|aarch64)
    arch=arm64
    ;;
*)
    arch=amd64
    ;;
esac

# You can authenticate by exporting the GITHUB_TOKEN in the environment
if [[ -z "${GITHUB_TOKEN}" ]]; then
    releases=$(curl -s "$release_url")
else
    releases=$(curl -s "$release_url" --header "Authorization: Bearer ${GITHUB_TOKEN}")
fi

if [[ $releases == *"API rate limit exceeded"* ]]; then
  echo "Github rate-limiter failed the request. Either authenticate or wait a couple of minutes."
  exit 1
fi

RELEASE_URL="$(find_release_url "$releases" "$opsys" "$arch")"

if [[ -z "$RELEASE_URL" ]]; then
  echo "Version $version does not exist or is not available for ${opsys}/${arch}."
  exit 1
fi

curl -sLO "$RELEASE_URL"

cp "./emergency-credentials-receive_${opsys}_${arch}" "$where/emergency-credentials-receive"
chmod +x "$where/emergency-credentials-receive"

popd >& /dev/null

"${where}emergency-credentials-receive" -h

echo "emergency-credentials-receive installed to ${where}emergency-credentials-receive"
