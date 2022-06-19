package main

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed webdata/*
var webdataFSWithPrefix embed.FS
var webdataFS fs.FS

func init() {
	var err error
	webdataFS, err = fs.Sub(webdataFSWithPrefix, "webdata")
	if err != nil {
		log.Fatal(err)
	}
}
