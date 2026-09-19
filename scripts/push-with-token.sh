#!/usr/bin/env bash
#
# Push finance_app to GitHub using a personal access token.
#
# The token is read from ~/.config/finance_app/github.env, which lives outside
# the repository so it cannot be committed. Care is taken to keep it out of
# three places it would otherwise leak into:
#
#   - argv, visible to any local process via `ps`: curl reads the auth header
#     from stdin via `-K -` instead of taking it as an argument.
#   - .git/config and the reflog: the remote URL stays credential-free and the
#     token is supplied through a one-shot credential helper.
#   - this script's output: nothing ever echoes the token.
#
# Usage:
#   ./scripts/push-with-token.sh            # push to an existing repo
#   ./scripts/push-with-token.sh --create   # also create the repo if missing
#
# With --create the repository is made PRIVATE. Pass --public to override, which
# publishes the code to anyone on the internet.

set -euo pipefail

readonly CONFIG="${FINANCE_APP_GITHUB_ENV:-$HOME/.config/finance_app/github.env}"
readonly OLD_OWNER="lamcheryl"
readonly API="https://api.github.com"

create_repo=false
visibility=private

while [ $# -gt 0 ]; do
	case "$1" in
	--create) create_repo=true ;;
	--public) visibility=public ;;
	--private) visibility=private ;;
	-h | --help)
		sed -n '2,24p' "$0" | sed 's/^#//'
		exit 0
		;;
	*)
		echo "error: unknown argument '$1'" >&2
		exit 2
		;;
	esac
	shift
done

die() {
	echo "error: $*" >&2
	exit 1
}

# --- Load the token -----------------------------------------------------------

[ -f "$CONFIG" ] || die "no config at $CONFIG. Ask Claude to recreate the template."

# Warn on loose permissions rather than silently using a world-readable secret.
perms="$(stat -f '%Lp' "$CONFIG" 2>/dev/null || stat -c '%a' "$CONFIG" 2>/dev/null || echo '')"
case "$perms" in
600 | 400 | '') ;;
*) echo "warning: $CONFIG is mode $perms; run: chmod 600 '$CONFIG'" >&2 ;;
esac

# shellcheck source=/dev/null
set -a
. "$CONFIG"
set +a

GITHUB_TOKEN="${GITHUB_TOKEN:-}"
GITHUB_USERNAME="${GITHUB_USERNAME:-}"
GITHUB_REPO="${GITHUB_REPO:-finance_app}"
[ -n "$GITHUB_REPO" ] || GITHUB_REPO=finance_app

if [ -z "$GITHUB_TOKEN" ]; then
	die "GITHUB_TOKEN is empty in $CONFIG. Open it, paste your token after the '=', and save."
fi
case "$GITHUB_TOKEN" in
*[[:space:]]*) die "GITHUB_TOKEN contains whitespace. Remove any quotes or trailing spaces." ;;
esac

# --- API helper ---------------------------------------------------------------

# gh_api METHOD PATH [BODY]
# Prints "<http_status>\n<body>". The auth header goes in via stdin so the token
# never appears in the process list.
gh_api() {
	local method="$1" path="$2" body="${3:-}"
	local -a args=(
		--silent --show-error --location
		--write-out '\n%{http_code}'
		--request "$method"
		--header 'Accept: application/vnd.github+json'
		--header 'X-GitHub-Api-Version: 2022-11-28'
	)
	if [ -n "$body" ]; then
		args+=(--header 'Content-Type: application/json' --data "$body")
	fi

	local out
	out="$(printf 'header = "Authorization: Bearer %s"\n' "$GITHUB_TOKEN" |
		curl -K - "${args[@]}" "${API}${path}")" || die "curl failed contacting the GitHub API"

	# Reorder to status-first so callers can read it off line 1.
	printf '%s\n' "${out##*$'\n'}"
	printf '%s\n' "${out%$'\n'*}"
}

json_field() {
	# Reads JSON on stdin, prints one top-level string field. Uses python3 so we
	# do not depend on jq being installed.
	python3 -c '
import json, sys
try:
    data = json.load(sys.stdin)
except Exception:
    sys.exit(1)
value = data.get(sys.argv[1]) if isinstance(data, dict) else None
if value is None:
    sys.exit(1)
print(value)
' "$1"
}

# --- 1. Verify the token and learn the account --------------------------------

echo "==> Verifying token against the GitHub API"

response="$(gh_api GET /user)"
status="$(printf '%s' "$response" | head -1)"
body="$(printf '%s' "$response" | tail -n +2)"

case "$status" in
200) ;;
401) die "GitHub rejected the token (401). It may be expired, revoked, or mistyped." ;;
403) die "GitHub returned 403. The token may lack the required permissions, or you are rate limited." ;;
*) die "unexpected status $status from GET /user" ;;
esac

login="$(printf '%s' "$body" | json_field login)" || die "could not read the account login from the API response"

if [ -n "$GITHUB_USERNAME" ] && [ "$GITHUB_USERNAME" != "$login" ]; then
	die "config says GITHUB_USERNAME=$GITHUB_USERNAME but the token belongs to $login"
fi
owner="$login"
echo "    authenticated as $owner"

# --- 2. Make sure the repository exists ---------------------------------------

echo "==> Checking github.com/$owner/$GITHUB_REPO"

response="$(gh_api GET "/repos/$owner/$GITHUB_REPO")"
status="$(printf '%s' "$response" | head -1)"

case "$status" in
200)
	echo "    found"
	;;
404)
	if [ "$create_repo" != true ]; then
		die "$owner/$GITHUB_REPO does not exist (or the token cannot see it).
       Create it at https://github.com/new — empty, no README and no license —
       or re-run with --create to have this script make it."
	fi
	echo "==> Creating $owner/$GITHUB_REPO ($visibility)"
	payload="$(python3 -c '
import json, sys
print(json.dumps({
    "name": sys.argv[1],
    "private": sys.argv[2] == "private",
    "description": "Personal finance tracker: Go API, TypeScript frontend",
    "has_wiki": False,
    "has_projects": False,
    "auto_init": False,
}))
' "$GITHUB_REPO" "$visibility")"

	response="$(gh_api POST /user/repos "$payload")"
	status="$(printf '%s' "$response" | head -1)"
	body="$(printf '%s' "$response" | tail -n +2)"
	case "$status" in
	201) echo "    created" ;;
	403) die "403 creating the repository. A fine-grained token cannot create repos; use a classic token with the 'repo' scope, or create it in the browser." ;;
	422) die "422 creating the repository. The name may already be taken: $(printf '%s' "$body" | head -c 400)" ;;
	*) die "unexpected status $status creating the repository" ;;
	esac
	;;
403)
	die "403 reading the repository. If this is a fine-grained token, check that finance_app is in its selected repositories and that Contents is set to Read and write."
	;;
*)
	die "unexpected status $status from GET /repos/$owner/$GITHUB_REPO"
	;;
esac

# --- 3. Rewrite the Go module path if the account differs ---------------------

cd "$(dirname "$0")/.."

if [ -n "$(git status --porcelain)" ]; then
	die "working tree has uncommitted changes; commit or stash before pushing"
fi

# go.mod is plain text and the sed pass below covers its module line too, so
# this needs no `go mod edit` and therefore no Go toolchain. Driving off what
# is actually in the tree rather than off the owner name keeps this
# idempotent: a second run finds nothing to change and goes straight to the
# push.
files="$(git grep -l "github.com/${OLD_OWNER}/finance_app" || true)"

if [ -n "$files" ] && [ "$owner" != "$OLD_OWNER" ]; then
	echo "==> Rewriting Go module path: $OLD_OWNER -> $owner"

	if sed --version >/dev/null 2>&1; then
		printf '%s\n' "$files" | xargs sed -i "s|github.com/${OLD_OWNER}/finance_app|github.com/${owner}/${GITHUB_REPO}|g"
	else
		printf '%s\n' "$files" | xargs sed -i '' "s|github.com/${OLD_OWNER}/finance_app|github.com/${owner}/${GITHUB_REPO}|g"
	fi

	if command -v go >/dev/null 2>&1; then
		echo "    verifying it still builds"
		(cd backend && go build ./... && go vet ./...) || die "the rewrite broke the build; inspect the diff before pushing"
	else
		echo "    (go not installed, skipping build check)" >&2
	fi

	git add -A
	git commit -q -m "Set module path to github.com/${owner}/${GITHUB_REPO}"
	echo "    committed"
else
	echo "==> Module path already points at github.com/${owner}/${GITHUB_REPO}"
fi

# --- 4. Push ------------------------------------------------------------------

remote_url="https://github.com/${owner}/${GITHUB_REPO}.git"

if git remote get-url origin >/dev/null 2>&1; then
	git remote set-url origin "$remote_url"
else
	git remote add origin "$remote_url"
fi
echo "==> origin is $remote_url"

# Refuse to overwrite existing history on the remote.
if git ls-remote --exit-code --heads origin main >/dev/null 2>&1; then
	echo "    note: origin already has a main branch; pushing as a fast-forward only"
fi

echo "==> Pushing main"

# The credential helper is passed with -c so it applies to this command only and
# is never written to .git/config. It reads the token from the environment
# rather than taking it as an argument.
export FINANCE_APP_TOKEN="$GITHUB_TOKEN"
helper='!f() { test "$1" = get && printf "username=x-access-token\npassword=%s\n" "$FINANCE_APP_TOKEN"; }; f'

git -c credential.helper= -c "credential.helper=$helper" push --verbose origin main

unset FINANCE_APP_TOKEN

echo
echo "Done: https://github.com/${owner}/${GITHUB_REPO}"
echo
echo "The token is still in $CONFIG."
echo "To remove it now:  rm '$CONFIG'"
echo "To revoke it:      https://github.com/settings/tokens"
