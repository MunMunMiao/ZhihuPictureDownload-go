build:
	GO111MODULE=on GOPROXY=https://goproxy.io CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags '-w -s' -o bin/zhihu-linux main.go
	GO111MODULE=on GOPROXY=https://goproxy.io CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -ldflags '-w -s' -o bin/zhihu-win.exe main.go
	GO111MODULE=on GOPROXY=https://goproxy.io CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -ldflags '-w -s' -o bin/zhihu-darwin main.go