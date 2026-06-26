#!/bin/bash

set -e

declare HYSTRIX_TAG

help() {
   printf "\n"
   printf "Usage: %s [-t tag]\n" "$0"
   printf "\t-t New Hystrix unreleased tag. Updates all submodule go.mod files with this tag.\n"
   exit 1 # Exit script after printing help
}

while getopts "t:" opt
do
   case "$opt" in
      t ) HYSTRIX_TAG="$OPTARG" ;;
      ? ) help ;; # Print help
   esac
done

declare -r SEMVER_REGEX="^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(\\-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"

validate_tag() {
    local tag_=$1
    if [[ "${tag_}" =~ ${SEMVER_REGEX} ]]; then
	    printf "%s is valid semver tag.\n" "${tag_}"
    else
	    printf "%s is not a valid semver tag.\n" "${tag_}"
	    return 1
    fi
}

# Print help in case parameter is empty
if [[ -z "$HYSTRIX_TAG" ]]
then
    printf "parameter '-t' must be specified.\n"
    help
fi


## Validate tags first
validate_tag "${HYSTRIX_TAG}" || exit $?
TAG_FOUND=$(git tag --list "${HYSTRIX_TAG}")
if [[ ${TAG_FOUND} = "${HYSTRIX_TAG}" ]] ; then
    printf "Tag %s already exists in this repo\n" "${HYSTRIX_TAG}"
    exit 1
fi

cd "$(dirname "$0")"

if ! git diff --quiet; then \
    printf "Working tree is not clean, can't proceed\n"
    git status
    git diff
    exit 1
fi

declare -r BRANCH_NAME=pre_release_${HYSTRIX_TAG}

patch_gomods() {
    local pkg_=$1
    local tag_=$2
    # now do the same for all the directories underneath
    PACKAGE_DIRS=$(find . -mindepth 2 -type f -name 'go.mod' -exec dirname {} \; | egrep -v 'tools' | sed 's|^\.\/||' | sort)
    # quote any '.' characters in the pkg name
    local quoted_pkg_=${pkg_//./\\.}
    for dir in $PACKAGE_DIRS; do
	    cp "${dir}/go.mod" "${dir}/go.mod.bak"
	    sed "s|${quoted_pkg_}\([^ ]*\) v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*[^0-9]*.*$|${pkg_}\1 ${tag_}|" "${dir}/go.mod.bak" >"${dir}/go.mod"
	    rm -f "${dir}/go.mod.bak"
    done
}

# branch off from existing branch master
git checkout -b "${BRANCH_NAME}" master

if [ -n "${HYSTRIX_TAG}" ]; then
    patch_gomods github.com/gojek/hystrix-go "${HYSTRIX_TAG}"
fi

# Run gomod.tidy to update go.sum
make gomod.tidy

# Add changes and commit.
git add .

declare COMMIT_MSG=""
COMMIT_MSG+="Releasing ${HYSTRIX_TAG}"
git commit -m "${COMMIT_MSG}"

printf "Now run following to verify the changes.\ngit diff master\n"
printf "\nThen push the changes to upstream\n"