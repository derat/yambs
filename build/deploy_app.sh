#!/bin/sh -e

usage=$(cat <<EOF
Usage: $0 [flags]... [gcloud args]...
Deploy the App Engine app to GCP and delete old versions.

Flags:
  -c  Short commit hash
  -p  GCP project ID to use
EOF
)

commit=unknown
project=
while getopts c:p: o; do
  case $o in
    c) commit="$OPTARG" ;;
    p) project="$OPTARG" ;;
    \?) echo "$usage" >&2 && exit 2 ;;
  esac
done
shift $(expr $OPTIND - 1)

if [ -z "$project" ]; then
  echo 'GCP project ID must be specified using -p' >&2
  exit 2
fi

# Inject a version string into app.yaml.
version="$(date --utc +%Y%m%d)-${commit}"
sed -i -r -e "s/^(\s*APP_VERSION:).*/\1 '${version}'/" app.yaml

# As of November 2021, 'beta' is required here to use App Engine bundled
# services (e.g. memcache) from the go115 runtime. Without it, the 'deploy'
# command prints 'WARNING: There is a dependency on App Engine APIs, but they
# are not enabled in your app.yaml. Set the app_engine_apis property.' even when
# the property is set.
#
# Surprisingly, --quiet only disables yes/no prompts (which we want) rather than
# suppressing output (which we don't want).
echo 'Deploying app...'
gcloud beta app --project="$project" --quiet deploy "$@"

# Clean up stale versions of the app so they aren't sitting around.
echo 'Deleting old versions...'
versions=$(
  gcloud app --project="$project" versions list |
    sed -nre 's/^default +([^ ]+) +0\.00 .*/\1/p'
)
if [ -n "$versions" ]; then
  gcloud app --project="$project" --quiet versions delete $versions
fi
