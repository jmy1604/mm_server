#!/bin/bash
set -x

cd ../bin
nohup env GOTRACEBACK=crash `pwd`/game_server -f `pwd`/../conf/game_server3.json 1>/dev/null 2>>gs3_err.log &
