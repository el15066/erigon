#!/bin/bash -xe

for i in {0..9}; do

make erigon

echo 3 | sudo tee /proc/sys/vm/drop_caches

sleep 15

./build/bin/erigon --datadir /media/route/sx8200/erigon/data/ --ethash.dagdir /media/route/sx8200/erigon/ethash/ &>>logz/run_$i.log || true

echo '418c418
< 	blockChan := make(chan *types.Block, '$i')
---
> 	blockChan := make(chan *types.Block, '$(($i+1))')
' | patch -s eth/stagedsync/stage_execute.go

sleep 5

done
