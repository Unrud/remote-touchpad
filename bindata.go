package main

//go:generate go-bindata -nometadata -o ./bindata.generated.go webdata/...
//go:generate gofmt -w ./bindata.generated.go

import "github.com/elazarl/go-bindata-assetfs"

func assetFS() *assetfs.AssetFS {
	return &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo, Prefix: "webdata"}
}
