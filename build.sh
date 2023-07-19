#! /bin/sh
export GOOS=linux
export GOARCH=amd64
exe_file=server
if [ -n "$1" ];then
    echo "build -o $1"
    exe_file=$1
fi
VERSION=$2
echo "version:${VERSION}"
mkdir -p bin/
go build -ldflags "-X main.Version=${VERSION}" -o ./bin/$exe_file ./main/*.go