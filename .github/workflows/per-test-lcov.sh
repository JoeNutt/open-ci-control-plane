#!/usr/bin/env bash
set -euo pipefail

readonly top_level_test='^Test[[:alnum:]_]*$'
readonly per_test_timeout=5m

go_root="$(go env GOROOT)"
readonly go_root

readonly package="$1"
readonly artifact="$2"
scratch="$(mktemp -d)"
readonly scratch
trap 'rm -rf "$scratch"' EXIT
readonly test_binary="$scratch/package.test"

main() {
  : >"$artifact"
  go test -c -cover -covermode=atomic -coverpkg=./... -o "$test_binary" "$package"
  if [[ ! -x "$test_binary" ]]; then
    echo "$package has no tests" >&2
    return 0
  fi
  local package_directory
  package_directory="$(go list -f '{{.Dir}}' "$package")"

  local test_list_file="$scratch/top-level-tests"
  top_level_tests "$package_directory" >"$test_list_file"

  local test
  while IFS= read -r test; do
    section_of "$package_directory" "$test" >>"$artifact"
  done <"$test_list_file"
}

top_level_tests() {
  local package_directory="$1"
  local listed_names_file="$scratch/listed-names"
  run_test_binary "$package_directory" "$scratch/listing-counters" -test.list '.*' >"$listed_names_file"
  refuse_unrunnable_names "$listed_names_file"
  grep -E "$top_level_test" "$listed_names_file" || [[ $? -eq 1 ]]
}

refuse_unrunnable_names() {
  local unrunnable
  unrunnable="$(grep -vE "$top_level_test" "$1" | tr '\n' ' ' || [[ $? -eq 1 ]])"
  if [[ -n "$unrunnable" ]]; then
    echo "$package lists names this collection does not run: $unrunnable" >&2
    return 1
  fi
}

section_of() {
  local package_directory="$1"
  local test="$2"
  local counters="$scratch/counters-$test"
  local profile="$scratch/$test.out"
  local unnamed_lcov="$scratch/$test.lcov"

  run_test_binary "$package_directory" "$counters" -test.run "^${test}\$" -test.timeout "$per_test_timeout" >/dev/null
  go tool covdata textfmt -i="$counters" -o="$profile"
  GOROOT="$go_root" gcov2lcov -infile "$profile" -outfile "$unnamed_lcov"
  exercised_records_of "$unnamed_lcov" "$test"
}

run_test_binary() {
  local package_directory="$1"
  local counters="$2"
  shift 2
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
