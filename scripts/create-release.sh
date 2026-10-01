#!/usr/bin/env bash
set -euo pipefail

# Create a GitHub release with auto-incremented version.
# Usage: create-release.sh [--publish]
#   --publish: Create a published release with auto-generated notes
#   No flag:   Create a draft release with placeholder body

publish=false
for arg in "$@"; do
	case "${arg}" in
	--publish) publish=true ;;
	*)
		echo "Unknown argument: ${arg}" >&2
		exit 1
		;;
	esac
done

# ── Get repo owner/repo from git remote ──────────────────────────────────

remote_url=$(git remote get-url origin)

# Support both HTTPS and SSH formats — strip trailing .git if present
remote_url="${remote_url%.git}"
if [[ ${remote_url} =~ ^https://github\.com/([^/]+)/([^/]+)$ ]]; then
	owner="${BASH_REMATCH[1]}"
	repo="${BASH_REMATCH[2]}"
elif [[ ${remote_url} =~ ^git@github\.com:([^/]+)/([^/]+)$ ]]; then
	owner="${BASH_REMATCH[1]}"
	repo="${BASH_REMATCH[2]}"
else
	echo "Unsupported remote URL format: ${remote_url}" >&2
	exit 1
fi

echo "Repository: ${owner}/${repo}"
if [[ ${publish} == true ]]; then
	echo "Mode: publish (with auto-generated notes)"
else
	echo "Mode: draft"
fi

# ── Get latest version tag ───────────────────────────────────────────────

latest_tag=""

# Try GitHub API first (latest release)
if api_tag=$(gh api "repos/${owner}/${repo}/releases/latest" --jq '.tag_name' 2>/dev/null); then
	latest_tag="${api_tag}"
fi

# Fall back to git tags if no release found
if [[ -z ${latest_tag} ]]; then
	latest_tag=$(git tag --list --sort=-version:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -n1 || true)
fi

if [[ -z ${latest_tag} ]]; then
	echo "No previous releases found. Please create an initial release manually." >&2
	exit 1
fi

echo "Latest release tag: ${latest_tag}"

# ── Parse and increment version ──────────────────────────────────────────

version_str="${latest_tag#v}"
if ! [[ ${version_str} =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
	echo "Invalid version format: ${latest_tag}" >&2
	exit 1
fi

major="${BASH_REMATCH[1]}"
minor="${BASH_REMATCH[2]}"
build="${BASH_REMATCH[3]}"

echo "Current version: v${major}.${minor}.${build}"

# Auto-increment: if build >= 10, bump minor and reset build
if [[ ${build} -ge 10 ]]; then
	minor=$((minor + 1))
	build=0
else
	build=$((build + 1))
fi

new_tag="v${major}.${minor}.${build}"
echo "New version: ${new_tag}"

# ── Create the release ───────────────────────────────────────────────────

gh_args=(
	"repos/${owner}/${repo}/releases"
	-f "tag_name=${new_tag}"
	-f "target_commitish=main"
	-f "name=Release ${new_tag}"
	-F "prerelease=false"
)

if [[ ${publish} == true ]]; then
	gh_args+=(-F "draft=false" -F "generate_release_notes=true")
else
	gh_args+=(-F "draft=true" -f "body=Release ${new_tag}

## Changes

- TODO: Add release notes")
fi

if ! release_url=$(gh api --method POST "${gh_args[@]}" --jq '.html_url'); then
	echo "Error creating release" >&2
	exit 1
fi

if [[ ${publish} == true ]]; then
	echo ""
	echo "Successfully published release!"
	echo "  Version: ${new_tag}"
	echo "  URL: ${release_url}"
else
	echo ""
	echo "Successfully created draft release!"
	echo "  Version: ${new_tag}"
	echo "  URL: ${release_url}"
	echo "  Don't forget to:"
	echo "  1. Edit the release notes"
	echo "  2. Publish the release when ready"
fi
