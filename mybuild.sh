#! /bin/sh

set -xe

exe_file=subpub
os=$1
arch=$2
ver=$3

if [ "x$os" = "x" ];then
    os="linux"
fi

if [ "x$arch" = "x" ];then
    arch="amd64"
fi

if [ x$ver = "x" ];then     
    buildVersion=`git describe --tags` #v2.0.1-17-g0d7b4df
    ver=`git describe --abbrev=0 --tags` #最新的tag:v2.0.1
else
    commitID=`git rev-parse HEAD`
    buildVersion= $ver-$commitID
fi

time=`date +%Y%m%d`
buildTime=`echo ${time:2}`
goversion=`go version|awk '{print $3}'`

echo "os:$os, arch:$arch, ver:$ver, buildVersion:$buildVersion, buildTime:$buildTime, goVersion:$goversion"

GOOS=$os GOARCH=$arch go build -o ${exe_file}_$os${arch}_$ver -ldflags \
"-s -w -X main.Version=$buildVersion -X main.BuildTime=$buildTime -X main.BuildGoVersion=$goversion" main/main.go
