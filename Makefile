PROJECT_NAME:=server
VERSION:=$(shell git describe --tags --always)

ImageName:=subpub_${PROJECT_NAME}:${VERSION}
ContainerName:=subpub_${PROJECT_NAME}
#ContainerName like subpub_server:v2.0.1-9-g696ef33

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

docker:
#@git pull
# @docker build -t ${ImageName} .
	make build;
	make image;
	@echo "docker build success"
	@container_id=$$(docker ps -a -f name=${ContainerName} -q); \
    if [ -n "$$container_id" ]; then \
		docker rm -f "$$container_id"; \
        echo "Container ${ContainerName} deleted"; \
    else \
        echo "Container ${ContainerName} not found"; \
    fi
	make run;
	@echo "docker start ${ContainerName} success"