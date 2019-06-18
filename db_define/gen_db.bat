set GOPATH=%cd%/../../../

md proto

cd ../tools
code_generator.exe -c ../db_define/login_db.json -d ../src/login_server -p ../db_define/proto/login_db.proto

cd ../third_party/protobuf
protoc.exe --go_out=../../src/login_server/login_db --proto_path=../../db_define/proto login_db.proto
cd ../../db_define
