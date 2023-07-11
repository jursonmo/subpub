PROJECT_NAME:=server
VERSION:=v2
ImageName:=subpub_${PROJECT_NAME}:${VERSION}
ContainerName:=subpub_${PROJECT_NAME}

.PHONY: image run stop build rm rmi clean

build:
	bash build.sh ${PROJECT_NAME}

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