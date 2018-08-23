OS=linux
ARCH=amd64

bin:
	docker build -f Dockerfile.artifacts -t remark42.bin .
	docker run -d --name=remark42.bin remark42.bin
	docker cp remark42.bin:/artifacts/remark42.$(OS)-$(ARCH) remark42
	docker rm -f remark42.bin

docker:
	docker build -t umputun/remark42 --build-arg SKIP_FRONTEND_TEST=true --build-arg SKIP_BACKEND_TEST=true .

deploy:
	docker build -f Dockerfile.artifacts -t remark42.bin .
	docker run -d --name=remark42.bin remark42.bin
	- @mkdir -p bin
	docker cp remark42.bin:/artifacts/remark42.linux-amd64.tar.gz bin/remark42.linux-amd64.tar.gz
	docker cp remark42.bin:/artifacts/remark42.linux-386.tar.gz bin/remark42.linux-386.tar.gz
	docker cp remark42.bin:/artifacts/remark42.linux-arm64.tar.gz bin/remark42.linux-arm64.tar.gz
	docker cp remark42.bin:/artifacts/remark42.darwin-amd64.tar.gz bin/remark42.darwin-amd64.tar.gz
	docker cp remark42.bin:/artifacts/remark42.windows-amd64.zip bin/remark42.windows-amd64.zip
	docker rm -f remark42.bin


