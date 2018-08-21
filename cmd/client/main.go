package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/glog"
	"github.com/maxmcd/collab/pkg/files"
)

var HOST = "http://localhost:8080"

func incorrectUse() {
	fmt.Println("Use with 'collab serve NAME' or 'collab receive NAME'")
}
func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()
	if len(os.Getenv("HOST")) != 0 {
		HOST = os.Getenv("HOST")
	}
	if len(os.Args) < 3 {
		incorrectUse()
		return
	}

	name := os.Args[2]
	switch command := os.Args[1]; command {
	case "serve":
		serve(name)
	case "receive":
		receive(name)
	default:
		incorrectUse()
	}
}

func serve(name string) {
	md := files.New(HOST, name)
	glog.Info("Reading files and uploading data, do not edit directory contents!")
	if err := md.ReadAllFiles(); err != nil {
		glog.Fatal(err)
	}
	if err := md.UploadChunks(); err != nil {
		glog.Fatal(err)
	}
	if err := md.UploadDirectory(); err != nil {
		glog.Fatal(err)
	}
	glog.Info("Completed! Access this working directory with: `collab receive " + name + "`")
	if err := md.WatchFiles(); err != nil {
		glog.Fatal(err)
	}
}

func receive(name string) {
	if fls, err := ioutil.ReadDir("./"); err != nil {
		glog.Fatal(err)
	} else if len(fls) > 0 {
		glog.Fatal("This is not an empty directory. Create" +
			" an empty directory and run the command again to " +
			"download the shared directory")
	}
	md := files.New(HOST, name)
	if err := md.FetchAllFiles(); err != nil {
		glog.Fatal(err)
	}
	if err := md.CreateFiles(); err != nil {
		glog.Fatal(err)
	}
	if err := md.WatchFiles(); err != nil {
		glog.Fatal(err)
	}
}
