#!/bin/bash -ue
cd $(dirname $0)/../..

# Build binaries
build_dir=$PWD/build
mkdir -p ${build_dir}

go generate ./...
go build -o ${build_dir} ./...

# Start server and wait for it to become ready
echo server starting
${build_dir}/server 2> >(sed "s/^/  /" ) 1> >(sed "s/^/  /" )  &
server_pid=$!

# Check that server is running after a second, otherwise stop
sleep 1
if ! kill -0 ${server_pid} 2> /dev/null; then
  echo "unable to start server" && exit 1
fi

# Stop server after test
trap 'kill ${server_pid} && echo server stopped' EXIT

# Run load test
echo client starting
${build_dir}/load_client 2> >(sed "s/^/  /" ) 1> >(sed "s/^/  /" )
echo client stopped
