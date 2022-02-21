package main

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed webdata/*
var webdataWithPrefix embed.FS
var webdata fs.FS

func init() {
	var err error
	webdata, err = fs.Sub(webdataWithPrefix, "webdata")
	if err != nil {
		log.Fatal(err)
	}
}
