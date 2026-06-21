
export CGO_ENABLED=0
export GOOS=windows
export GOARCH=amd64

go build -ldflags "-s -w" -o ../image-transmit.exe
