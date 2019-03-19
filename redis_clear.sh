#!/bin/bash

REDIS_HOSTNAME="127.0.0.1"
REDIS_PORT="6379"

redis-cli -h $REDIS_HOSTNAME -p $REDIS_PORT del "mm:game_server:google_pay" "mm:game_server:apple_pay" "mm:share_data:uid_player_list" "mm:game_server:uid_token_key"
