IMAGE_NAME :=hello-svc
VERSION :=v1.0
CONTAINER_NAME :=hello-svc

build:
	docker build . -t ${IMAGE_NAME}:${VERSION} -f Dockerfile
run:
	docker run -itd --name ${CONTAINER_NAME} -p 8090:8090 -p 50051:50051 ${IMAGE_NAME}:${VERSION}

rerun: remove run

rebuild-run: build rerun

stop:
	docker stop ${CONTAINER_NAME}

restart:
	docker restart ${CONTAINER_NAME}

remove:
	docker rm -f ${CONTAINER_NAME}
exec:
	docker exec -it ${CONTAINER_NAME} /bin/bash
logs:
	docker logs ${CONTAINER_NAME} -f

gen: gen-pb gen-node

gen-pb:
	sh bin/go-generate.sh

gen-node:
	sh bin/nodejs-gen.sh
