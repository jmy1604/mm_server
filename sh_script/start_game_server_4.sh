#!/bin/bash
set -x

cd ../bin
nohup env GOTRACEBACK=crash `pwd`/game_server -f `pwd`/../conf/game_server4.json 1>/dev/null 2>>gs4_err.log &
