export GOPATH=$(pwd)/../../..
export PATH=$PATH:$GOPATH/bin

go get -u -v -t github.com/golang/protobuf/protoc-gen-go

cd ../db_define
mkdir -p proto

cd ../tools
./code_generator -c ../db_define/login_db.json -d ../src/login_server -p ../db_define/proto/login_db.proto

cd ../third_party/protobuf
./protoc --go_out=../../src/login_server/login_db --proto_path=../../db_define/proto login_db.proto
cd ../../db_define