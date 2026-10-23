#!/usr/bin/env bash
set -euo pipefail

readonly top_level_test='^Test[[:alnum:]_]*$'
readonly per_test_timeout=5m

go_root="$(go env GOROOT)"
readonly go_root

package="$1"
artifact="$2"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT
test_binary="$scratch/package.test"
package_directory=""

main() {
  : >"$artifact"
  go test -c -cover -covermode=atomic -coverpkg=./... -o "$test_binary" "$package"
  if [[ ! -x "$test_binary" ]]; then
    echo "$package has no tests" >&2
    return 0
  fi
  package_directory="$(go list -f '{{.Dir}}' "$package")"

  local tests="$scratch/tests" test
  top_level_tests >"$tests"
  while IFS= read -r test; do
    section_of "$test" >>"$artifact"
  done <"$tests"
}

top_level_tests() {
  local listing="$scratch/listing"
  run_test_binary "$scratch/listing-counters" -test.list '.*' >"$listing"
  grep -E "$top_level_test" "$listing" || [[ $? -eq 1 ]]
}

section_of() {
  local test="$1"
  local counters="$scratch/counters-$test"
  local profile="$scratch/$test.out"
  local unnamed_lcov="$scratch/$test.lcov"

  run_test_binary "$counters" -test.run "^${test}\$" -test.timeout "$per_test_timeout" >/dev/null
  go tool covdata textfmt -i="$counters" -o="$profile"
  GOROOT="$go_root" gcov2lcov -infile "$profile" -outfile "$unnamed_lcov"
  exercised_records_of "$unnamed_lcov" "$test"
}

run_test_binary() {
  local counters="$1"
  shift
  mkdir -p "$counters"
  (cd "$package_directory" && GOCOVERDIR="$counters" "$test_binary" -test.gocoverdir="$counters" "$@" </dev/null)
}

exercised_records_of() {
  awk -v test="$2" '
    /^TN:/ { next }
    /^SF:/ { record = $0 "\n"; isExercised = 0; next }
    /^DA:/ {
      split(substr($0, length("DA:") + 1), lineData, ",")
      if (lineData[2] > 0) { record = record $0 "\n"; isExercised = 1 }
      next
    }
    /^end_of_record$/ {
      if (!isExercised) { next }
      if (!isNamed) { printf "TN:%s\n", test; isNamed = 1 }
      printf "%s%s\n", record, $0
      next
    }
    END { if (!isNamed) { printf "%s executed no line\n", test > "/dev/stderr" } }
  ' "$1"
}

main
