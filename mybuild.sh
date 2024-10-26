#! /bin/sh

set -xe

appName=subpub
os=$1
arch=$2
ver=$3

if [ "x$os" = "x" ];then
    os="linux"
fi

if [ "x$arch" = "x" ];then
    arch="amd64"
fi

# buildVersion 比 ver 多一个commitId
if [ x$ver = "x" ];then     
    buildVersion=`git describe --tags` #v2.0.1-17-g0d7b4df
    ver=`git describe --abbrev=0 --tags` #最新的tag:v2.0.1
else
    commitID=`git rev-parse HEAD`
    buildVersion= $ver-$commitID
fi

#GIT_REVISION=$(git rev-parse --short HEAD)
GIT_BRANCH=$(git name-rev --name-only HEAD)

time=`date +%Y%m%d`
buildTime=`echo ${time:2}`
goversion=`go version|awk '{print $3}'`

echo "os:$os, arch:$arch, ver:$ver, buildVersion:$buildVersion, buildTime:$buildTime, goVersion:$goversion"

#程序名称展示app名称,os和arch,版本号(tag), 程序运行还另外展示编译时commitId,编译时间,go版本,git分支
GOOS=$os GOARCH=$arch go build -o ${appName}_$os${arch}_$ver -ldflags \
"-s -w \
-X 'main.AppName=${appName}' \
-X 'main.AppVersion=${ver}'  \
-X main.BuildVersion=$buildVersion \
-X main.BuildTime=$buildTime \
-X main.BuildGoVersion=$goversion \
-X main.BuildGitBranch=$GIT_BRANCH " main/main.go

