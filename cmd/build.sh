 export CGO_ENABLED=0
export GOOS=linux
export GOARCH=arm64

go build -ldflags "-s -w" -o ../image-transmit.arm64
#upx image-transmit.arm64

export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64

go build -ldflags "-s -w" -o ../image-transmit.amd64
#upx image-transmit.amd64
