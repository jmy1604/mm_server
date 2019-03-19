#!/bin/bash
mkdir mm_server
cd mm_server
mkdir bin

cp ../bin/center_server ./bin
cp ../bin/login_server ./bin
cp ../bin/game_server ./bin
cp ../bin/rpc_server ./bin
cp ../bin/test_client ./bin

mkdir conf
cp -r ../conf/template ./conf 
cp -r ../game_data ./
cp -r ../sh_script ./

cp ../*.sh ./

cd ../

tar -czvf mm_server.tar.gz mm_server
rm -fr mm_server
