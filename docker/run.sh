#!/bin/bash

if [[ $# < 3 ]] ; then
    echo 'Usage: '$0' -host=<HOST> -ssh-user=<USER> -ssh-password=<PASSWD> [-countrytz=<TZ>] [-log-level=<LOGLEVEL>]' 
    exit 1
fi

ARGS=$1' '$2' '$3
if [[ $# > 3 ]] ; then
	ARGS=$ARGS' '$4
fi
if [[ $# > 4 ]] ; then
	ARGS=$ARGS' '$5
fi

ID=$(docker run --rm -d -p 9100 spiros-atos/torque_exporter $ARGS)

# Get dynamic port in host
PORT=$(docker ps --no-trunc|grep $ID|sed 's/.*0.0.0.0://g'|sed 's/->.*//g')
#PORT=$(docker-compose port web 80) #Only if using docker-compose, This will give us something like 0.0.0.0:32790.

echo $ID $PORT
