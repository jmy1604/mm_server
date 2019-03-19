#!/bin/bash
cd ../bin
export cur_dir=`pwd`
export cur_server_id=`ps aux | grep game_server | grep $cur_dir | awk 'NR==1{print $2}'`
if [ -z $cur_server_id ] ; then
	echo "cur_game_server not running"
	exit 0
else
	echo "cur_game_server id is $cur_server_id"
fi

kill -15 $cur_server_id

while [[ $cur_server_id != "" ]]
do
	export cur_server_id=`ps aux | grep game_server | grep $cur_dir | awk 'NR==1{print $2}'`
	if [ -z $cur_server_id ] ; then
        	echo "close_game_server ok"
	else
		kill -15 $cur_server_id
        	echo "wait game_server closing"
	fi

	sleep 1s
done
