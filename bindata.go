package main

//go:generate go-bindata-assetfs -nometadata -o ./bindata.generated.go webdata/...
//go:generate gofmt -w ./bindata.generated.go

import "net/http"

func fixedAssetFS() http.FileSystem {
	fs := assetFS()
	fs.AssetInfo = AssetInfo
	return fs
}
