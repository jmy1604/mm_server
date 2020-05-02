#!/bin/bash
export GO111MODULE=on
export GOPROXY=https://goproxy.io
export GOPATH=$(pwd)/../../..
set -x
go build -i -o ../bin/center_server mm_server/src/center_server
go build -i -o ../bin/login_server mm_server/src/login_server
go build -i -o ../bin/game_server mm_server/src/game_server
go build -i -o ../bin/rpc_server mm_server/src/rpc_server
go build -i -o ../bin/test_client mm_server/src/test_client
