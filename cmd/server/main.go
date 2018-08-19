package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/maxmcd/collab/pkg/files"
)

func chunkName(sha string) string {
	return "chunk-" + sha
}

var DIRECTORIES = map[string][]files.File{}

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()

	r := mux.NewRouter()
	r.HandleFunc("/chunk/{sha}", ChunkHandler).Methods("GET")
	r.HandleFunc("/chunk/{sha}", ChunkCheckHandler).Methods("HEAD")
	r.HandleFunc("/chunk/{sha}", ChunkCreateHandler).Methods("POST")
	r.HandleFunc("/directory/{name}", DirectoryHandler).Methods("GET")
	r.HandleFunc("/directory/{name}", DirectoryCreateHandler).Methods("POST")
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)

	port := ":8080"
	s := &http.Server{
		Addr:           port,
		Handler:        loggedRouter,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	glog.Infof("Broadcasting on port %s", port)
	log.Fatal(s.ListenAndServe())
}

func httpErr(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	glog.Error(err)
	fmt.Fprintf(w, err.Error())
}

func DirectoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	if _, ok := DIRECTORIES[name]; ok == false {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	allFiles := DIRECTORIES[name]
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(allFiles)
	if err != nil {
		httpErr(err, w)
		return
	}
}

func DirectoryCreateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	if _, ok := DIRECTORIES[name]; ok {
		w.WriteHeader(http.StatusConflict)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpErr(err, w)
		return
	}
	var allFiles []files.File
	if err := json.Unmarshal(body, &allFiles); err != nil {
		httpErr(err, w)
		return
	}
	DIRECTORIES[name] = allFiles
	w.WriteHeader(http.StatusCreated)
}

func ChunkCreateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sha := vars["sha"]
	// TODO: ensure sha is correct length/format
	if _, err := os.Stat(chunkName(sha)); os.IsNotExist(err) {
		file, err := os.Create(chunkName(sha))
		if err != nil {
			httpErr(err, w)
			return
		}
		// TODO: ensure body is < 4mb?
		if _, err := io.Copy(file, r.Body); err != nil {
			httpErr(err, w)
			return
		}
		if err := file.Close(); err != nil {
			httpErr(err, w)
			// TODO: rm file?
			return
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	} else if err != nil {
		httpErr(err, w)
	} else {
		w.WriteHeader(http.StatusNotModified)
	}
}

func ChunkCheckHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sha := vars["sha"]
	if _, err := os.Stat(chunkName(sha)); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		httpErr(err, w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func ChunkHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sha := vars["sha"]
	if _, err := os.Stat(chunkName(sha)); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		httpErr(err, w)
		return
	}
	f, err := os.Open(chunkName(sha))
	if err != nil {
		httpErr(err, w)
		return
	}
	if _, err := io.Copy(w, f); err != nil {
		httpErr(err, w)
		return
	}
	f.Close()
}
