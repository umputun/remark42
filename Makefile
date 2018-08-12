OS=linux
ARCH=amd64

bin:
	docker build -f Dockerfile.artifacts -t remark42.bin .
	docker run -d --name=remark42.bin remark42.bin
	docker cp remark42.bin:/artifacts/remark42.$(OS)-$(ARCH) remark42
	docker rm -f remark42.bin

docker:
	docker build -t remark42 --build-arg SKIP_FRONTEND_TEST=true --build-arg SKIP_BACKEND_TEST=true .
