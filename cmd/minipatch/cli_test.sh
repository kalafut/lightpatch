#!/usr/bin/env bash
set -euo pipefail

CMD=./minipatch
TD=../../testdata

for t in simple unicode angular
do

  # Test against known-good patch files
  if ! ($CMD apply $TD/${t}_in $TD/$t.patch | cmp -s $TD/${t}_out); then
    echo Failed apply test: ${t}; exit 1
  fi

  # Test against self-generated patch files
  $CMD make $TD/${t}_in $TD/${t}_out > "$TMPDIR/test.patch"
  if ! ($CMD apply $TD/${t}_in "$TMPDIR/test.patch" | cmp -s $TD/${t}_out); then
    echo Failed make/apply test: ${t}; exit 1
  fi

done

# Test CLI timeout
head -c 500000 /dev/urandom > "$TMPDIR/random_in"
head -c 500000 /dev/urandom > "$TMPDIR/random_out"

$CMD make --t 1s "$TMPDIR/random_in" "$TMPDIR/random_out" > "$TMPDIR/random.patch"
if ! ($CMD apply "$TMPDIR/random_in" "$TMPDIR/random.patch" | cmp -s "$TMPDIR/random_out"); then
  echo Failed random test; exit 1
fi

echo All test completed successfully