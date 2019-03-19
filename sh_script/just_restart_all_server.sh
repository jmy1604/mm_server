#!/bin/bash
set -x

bash ./kill_all_server.sh

sleep 1s

ulimit -c unlimited

cd ../bin
nohup env GOTRACEBACK=crash `pwd`/center_server 1>/dev/null 2>>cs_err.log &
sleep 1s
nohup env GOTRACEBACK=crash `pwd`/rpc_server 1>/dev/null 2>>rs_err.log &
sleep 1s
nohup env GOTRACEBACK=crash `pwd`/game_server 1>/dev/null 2>>gs_err.log &
sleep 1s 
nohup env GOTRACEBACK=crash `pwd`/game_server -f `pwd`/../conf/game_server2.json 1>/dev/null 2>>gs2_err.log &
sleep 1s
nohup env GOTRACEBACK=crash `pwd`/login_server 1>/dev/null 2>>ls_err.log &
