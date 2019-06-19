#!/bin/bash
set -x

cd ../bin
nohup env GOTRACEBACK=crash `pwd`/game_server -f `pwd`/../conf/game_server2.json 1>/dev/null 2>>gs2_err.log &
