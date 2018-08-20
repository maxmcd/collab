package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()

	r := mux.NewRouter()
	r.HandleFunc("/chunk/{sha}", ChunkHandler).Methods("GET")
	r.HandleFunc("/chunk/{sha}", ChunkCheckHandler).Methods("HEAD")
	r.HandleFunc("/chunk/{sha}", ChunkCreateHandler).Methods("POST")
	r.HandleFunc("/directory/{name}", DirectoryHandler).Methods("GET")
	r.HandleFunc("/directory/{name}", DirectoryCreateHandler).Methods("POST")
	r.HandleFunc("/events/{name}", WebsocketHandler)
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
