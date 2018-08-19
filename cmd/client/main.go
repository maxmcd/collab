package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/glog"
	"github.com/maxmcd/collab/pkg/files"
)

const HOST = "http://localhost:8080"

func incorrectUse() {
	fmt.Println("'collab serve NAME' or 'collab receive NAME'")
}
func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()

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
	allFiles, err := files.ReadAllFiles()
	if err != nil {
		glog.Fatal(err)
	}
	allFiles, err = files.UploadChunks(HOST, allFiles)
	if err != nil {
		glog.Fatal(err)
	}
	if err := files.UploadDirectory(HOST, name, allFiles); err != nil {
		glog.Fatal(err)
	}
	if err := files.WatchFiles(allFiles); err != nil {
		glog.Fatal(err)
	}
}

func receive(name string) {
	fls, err := ioutil.ReadDir("./")
	if err != nil {
		glog.Fatal(err)
	}
	if len(fls) > 0 {
		fmt.Println("This is not an empty directory. Create" +
			" an empty directory and run the command again to " +
			"download the shared directory")
		return
	}
	allFiles, err := files.FetchAllFiles(HOST, name)
	if err != nil {
		glog.Fatal(err)
	}
	if err := files.CreateFiles(HOST, allFiles); err != nil {
		glog.Fatal(err)
	}
	if err := files.WatchFiles(allFiles); err != nil {
		glog.Fatal(err)
	}
}
