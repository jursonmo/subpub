#! /bin/sh

set -xe

os=$1
arch=$2
ver=$3

if [ "x$os" = "x" ];then
    os="linux"
fi

if [ "x$arch" = "x" ];then
    arch="amd64"
fi


commitID=`git rev-parse HEAD`
time=`date +%Y%m%d`
buildTime=`echo ${time:2}`

#version with commitID and buildTime
buildVersion="$ver.$commitID.$buildTime"

if [ x$ver = "x" ];then     
    buildVersion=`git describe --tags`
    ver=`git describe --abbrev=0 --tags` #最新的tag
fi

echo "os:$os, arch:$arch, ver:$ver, buildVersion:$buildVersion"

GOOS=$os GOARCH=$arch go build -o $exe_file_$os${arch}_$ver -ldflags \
"-s -w -X main.version=$buildVersion" main.go
