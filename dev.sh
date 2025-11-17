#!/bin/bash

set -e -u -o pipefail

pushd web

pnpm build || exit 1

popd

go run main.go admin.go api.go guestbook.go anonymousMessages.go
