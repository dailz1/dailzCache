#!/bin/bash

lsof -i:8001,8002,8003,9999 |grep TCP | awk '{print $2}' | xargs kill -9
#trap "rm server;kill 0" EXIT

go build -o server
./server -port=8001 &
./server -port=8002 &
./server -port=8003 -api=1 &

sleep 2
echo ">>> start test"
wait

