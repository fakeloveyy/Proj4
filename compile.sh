#!/bin/bash

cd $(dirname $0)
export GOPATH=`pwd`
go install paxos/paxosutility
go install paxos
go install kvpaxos
mkdir bin
cd bin
go build start_server
go build stop_server
go build kvclient