#!/bin/bash

cd ../bin
export cur_dir=`pwd`

export cur_server_id=`ps aux | grep game_server | grep game_server4.json | grep $cur_dir | awk 'NR==1{print $2}'`
if [ -z $cur_server_id ] ; then
	echo "game_server_4 not running"
	exit 0
else
	echo "game_server_4 id is $cur_server_id"
fi

kill -15 $cur_server_id

while [[ $cur_server_id != "" ]]
do
	export cur_server_id=`ps aux | grep game_server  | grep game_server4.json | grep $cur_dir | awk 'NR==1{print $2}'`
	if [ -z $cur_server_id ] ; then
        	echo "close game_server_4 ok"
	else
		kill -15 $cur_server_id
        	echo "wait game_server_4 closing"
	fi

	sleep 1s
done
