#!/bin/sh
# Writes data/lastmod.json: for every file under content/, the date of the last commit that
# changed it on the checked-out branch. The image is built from site/ alone, where hugo has no
# git history to read, so the workflow runs this before the build and the templates take a
# page's date from the file.

set -eu

cd "$(dirname "$0")/.."

# an assignment, so that a git that cannot read the repository stops the script here
shallow=$(git rev-parse --is-shallow-repository)
if [ "$shallow" = "true" ]; then
	echo "page-dates: the clone is shallow, every page would get the date of its only commit; fetch the full history first" >&2
	exit 1
fi

out=data/lastmod.json
list=$(mktemp)
tmp=$(mktemp)
trap 'rm -f "$list" "$tmp"' EXIT

# quotePath off: git would otherwise write a name with a non-ASCII letter quoted and escaped.
# A name it still quotes holds a quote, a backslash or a control character, which the JSON
# below could not carry either
git -c core.quotePath=false ls-files -- content >"$list"
if grep -q '^"' "$list"; then
	echo "page-dates: a file under content/ has a quote, a backslash or a control character in its name:" >&2
	grep '^"' "$list" >&2
	exit 1
fi

sep=''
printf '{' >"$tmp"
# git lists the files in a fixed order, so the loop reads the list as it is
while IFS= read -r file; do
	# literal: a "[" or "*" in a name is not a pattern. first-parent: the date the change
	# reached this branch, not the date of a commit on a branch merged later. showSignature off:
	# with it on in the user's config, git prints the signature check before the date
	date=$(git -c log.showSignature=false --literal-pathspecs log -1 --first-parent --format=%cI -- "$file")
	case "$date" in
	[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9][+-][0-9][0-9]:[0-9][0-9]) ;;
	[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9]Z) ;;
	'')
		echo "page-dates: no commit found for $file" >&2
		exit 1
		;;
	*)
		echo "page-dates: git gave no date for $file but: $date" >&2
		exit 1
		;;
	esac
	printf '%s\n\t"%s": "%s"' "$sep" "${file#content/}" "$date" >>"$tmp"
	sep=','
done <"$list"
printf '\n}\n' >>"$tmp"

mv "$tmp" "$out"
echo "page-dates: wrote $(grep -c '": "' "$out") dates to site/$out"
