#! /usr/bin/env sh
# This script is a workaround to this issue: https://github.com/99designs/gqlgen/issues/1171
# Until they make it possible to use a custom errors package, we run this replace after gqlgen generate.

if find . -name "*.resolvers.go" -exec false {} +
then
  echo 'no files found'
  exit 0
fi

# Remove blank lines from imports to let gofmt sort all import lines consistently
find . -name "*.resolvers.go" -exec sed -i '' -e '
  /^import/,/)/ {
    /^$/ d
  }
' {} +

# Run gofmt, replacing native "errors" pkg with out own
find . -name "*.resolvers.go" -exec gofmt -w -r '"errors" -> "github.com/otterize/intents-operator/src/shared/errors"' {} +