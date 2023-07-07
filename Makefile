PROJECT_NAME:=server
VERSION:=v2
ImageName:=subpub_${PROJECT_NAME}:${VERSION}


.PHONY: image run stop build clean

build:
	bash build.sh ${PROJECT_NAME}

image:
	docker build -t ${ImageName} .

run:
	docker run  -itd  --name subpub_${PROJECT_NAME} \
	-p 8000:8000 ${ImageName}

stop:
	docker stop subpub_${PROJECT_NAME}

clean:
	rm -rf ./bin/${PROJECT_NAME} 