set -euo pipefail

[ -d /halfpipe-cache ] && export GOPATH="/halfpipe-cache/go"

go build gmapper.go

./gmapper