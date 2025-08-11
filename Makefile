# make还能用来编译打包成docker副本，使用make docker即可重新构建成docker副本
# tmd windows没有内置make
# docker的创建时间只有副本变了才会改
.PHONY: docker
docker:
	@rm webook || true
	@$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -buildvcs=false -tags=k8s -o webook .
	@docker rmi -f waterbucket/webook:v0.0.1
	@docker build -t waterbucket/webook:v0.0.1 .