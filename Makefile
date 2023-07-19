PROJECT_NAME:=server
VERSION:=$(shell git describe --tags --always)

ImageName:=subpub_${PROJECT_NAME}:${VERSION}
ContainerName:=subpub_${PROJECT_NAME}

.PHONY: image run stop build rm rmi clean

build:
	echo "build version:${VERSION}"
	bash build.sh ${PROJECT_NAME} ${VERSION}

image:
	docker build -t ${ImageName} .

run:
	docker run  -itd  --name ${ContainerName} \
	-p 8000:8000 ${ImageName}

stop:
	docker stop ${ContainerName}

rm:
	docker rm ${ContainerName}

rmi:
	echo "docker rmi ${ImageName}"
	docker rmi ${ImageName}

clean:
	rm -rf ./bin/${PROJECT_NAME} 