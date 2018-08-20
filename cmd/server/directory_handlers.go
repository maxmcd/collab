package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/maxmcd/collab/pkg/files"
)

// TODO: mutex
var DIRECTORIES = map[string][]files.File{}

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
