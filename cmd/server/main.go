package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func chunkName(sha string) string {
	return "chunk-" + sha
}

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()

	r := mux.NewRouter()
	r.HandleFunc("/chunk/{sha}", ChunkHandler).Methods("GET")
	r.HandleFunc("/chunk/{sha}", ChunkHeadHandler).Methods("HEAD")
	r.HandleFunc("/chunk/{sha}", ChunkPostHandler).Methods("POST")
	// r.HandleFunc("/directory/{name}", DirectoryHandler).Methods("GET")
	// r.HandleFunc("/directory/{name}", DirectoryHandler).Methods("POST")
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

func checkErr(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	glog.Error(err)
	fmt.Fprintf(w, err.Error())
}

func ChunkPostHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sha := vars["sha"]
	// TODO: ensure sha is correct length/format
	if _, err := os.Stat(chunkName(sha)); os.IsNotExist(err) {
		file, err := os.Create(chunkName(sha))
		if err != nil {
			checkErr(err, w)
			return
		}
		// TODO: ensure body is < 4kb?
		if _, err := io.Copy(file, r.Body); err != nil {
			checkErr(err, w)
			return
		}
		if err := file.Close(); err != nil {
			checkErr(err, w)
			// TODO: rm file?
			return
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	} else if err != nil {
		checkErr(err, w)
	} else {
		w.WriteHeader(http.StatusNotModified)
	}
}

func ChunkHeadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sha := vars["sha"]
	if _, err := os.Stat(chunkName(sha)); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		checkErr(err, w)
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
		checkErr(err, w)
		return
	}
	f, err := os.Open(chunkName(sha))
	if err != nil {
		checkErr(err, w)
		return
	}
	if _, err := io.Copy(w, f); err != nil {
		checkErr(err, w)
		return
	}
	f.Close()
}
