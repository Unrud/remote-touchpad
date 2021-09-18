package main

//go:generate go-bindata-assetfs -mode 0664 -modtime 1 -o ./bindata.generated.go webdata/...
//go:generate gofmt -w ./bindata.generated.go
