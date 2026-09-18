#!/usr/bin/env bash
#
# Point this repo at your GitHub account and push.
#
# The Go module path is baked into every import, so changing the account name
# means rewriting it across the tree. This does that, sets the remote, and
# pushes, in that order.
#
# Usage:
#   ./scripts/push-to-github.sh <github-username> [ssh|https]
#
# Requires that you can already authenticate to GitHub: an SSH key loaded for
# the ssh remote, or a credential helper / personal access token for https.

set -euo pipefail

readonly OLD_OWNER="lamcheryl"
readonly REPO="finance_app"

usage() {
	echo "usage: $0 <github-username> [ssh|https]" >&2
	exit 2
}

readonly OWNER="${1:-}"
readonly PROTO="${2:-ssh}"

[ -n "$OWNER" ] || usage
case "$PROTO" in
ssh | https) ;;
*)
	echo "error: protocol must be ssh or https, got '$PROTO'" >&2
	usage
	;;
esac

cd "$(dirname "$0")/.."

# Refuse to push a dirty tree: whatever is uncommitted would silently not go up.
if [ -n "$(git status --porcelain)" ]; then
	echo "error: working tree has uncommitted changes. Commit or stash first:" >&2
	git status --short >&2
	exit 1
fi

# --- 1. Rewrite the Go module path, if it is not already yours ----------------

if [ "$OWNER" != "$OLD_OWNER" ]; then
	echo "==> Rewriting module path: $OLD_OWNER -> $OWNER"

	(cd backend && go mod edit -module "github.com/${OWNER}/${REPO}/backend")

	# Every import of the old path, plus the README's note about it.
	files="$(git grep -l "github.com/${OLD_OWNER}/${REPO}" || true)"
	if [ -n "$files" ]; then
		# BSD sed on macOS needs the empty -i argument; GNU sed does not accept it.
		if sed --version >/dev/null 2>&1; then
			printf '%s\n' "$files" | xargs sed -i "s|github.com/${OLD_OWNER}/${REPO}|github.com/${OWNER}/${REPO}|g"
		else
			printf '%s\n' "$files" | xargs sed -i '' "s|github.com/${OLD_OWNER}/${REPO}|github.com/${OWNER}/${REPO}|g"
		fi
	fi

	if command -v go >/dev/null 2>&1; then
		echo "==> Verifying the rewrite compiles"
		(cd backend && go build ./... && go vet ./...)
	else
		echo "    (go not installed; skipping build check)" >&2
	fi

	git add -A
	git commit -q -m "Set module path to github.com/${OWNER}/${REPO}"
	echo "    committed module path change"
fi

# --- 2. Set the remote --------------------------------------------------------

if [ "$PROTO" = "ssh" ]; then
	url="git@github.com:${OWNER}/${REPO}.git"
else
	url="https://github.com/${OWNER}/${REPO}.git"
fi

if git remote get-url origin >/dev/null 2>&1; then
	echo "==> Updating origin -> $url"
	git remote set-url origin "$url"
else
	echo "==> Adding origin -> $url"
	git remote add origin "$url"
fi

# --- 3. Push ------------------------------------------------------------------

echo "==> Pushing main to origin"
git push -u origin main

echo
echo "Done. https://github.com/${OWNER}/${REPO}"
