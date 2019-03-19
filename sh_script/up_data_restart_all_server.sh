#!/bin/bash
set -x
bash ./kill_all_server.sh

sleep 5s

cd ../bin
nohup `pwd`/center_server &
sleep 1s
nohup `pwd`/rpc_server &
sleep 1s
nohup `pwd`/game_server &
sleep 1s 
nohup `pwd`/game_server -f `pwd`/../conf/game_server2.cfg &
sleep 1s
nohup `pwd`/login_server &


