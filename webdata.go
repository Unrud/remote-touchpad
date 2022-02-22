package main

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed webdata/*
var webdataFSWithPrefix embed.FS
var WebdataFS fs.FS

func init() {
	var err error
	WebdataFS, err = fs.Sub(webdataFSWithPrefix, "webdata")
	if err != nil {
		log.Fatal(err)
	}
}
