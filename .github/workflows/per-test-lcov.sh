#!/usr/bin/env bash
set -euo pipefail

package="$1"
artifact="$2"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

binary="$work/package.test"
go test -c -cover -covermode=atomic -coverpkg=./... -o "$binary" "$package"
: >"$artifact"
[[ -x "$binary" ]] || exit 0

directory="$(go list -f '{{.Dir}}' "$package")"
mkdir -p "$work/listing"
tests="$(cd "$directory" && "$binary" -test.gocoverdir="$work/listing" -test.list '.*' | grep -E '^Test[[:alnum:]_]*$' || true)"

for test in $tests; do
  counters="$work/counters-$test"
  mkdir -p "$counters"
  (cd "$directory" && "$binary" -test.gocoverdir="$counters" -test.run "^${test}\$" >/dev/null)
  go tool covdata textfmt -i="$counters" -o="$work/$test.out"
  gcov2lcov -infile "$work/$test.out" -outfile "$work/$test.lcov"
  awk -v test="$test" '
    /^TN:/ { next }
    /^SF:/ { record = $0 "\n"; exercised = 0; next }
    /^DA:/ { split(substr($0, 4), field, ","); if (field[2] > 0) { record = record $0 "\n"; exercised = 1 }; next }
    /^end_of_record$/ { if (exercised) { if (!named) { printf "TN:%s\n", test; named = 1 }; printf "%s%s\n", record, $0 }; next }
  ' "$work/$test.lcov" >>"$artifact"
done
