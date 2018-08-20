package main

import (
	"io"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func chunkName(sha string) string {
	return "chunk-" + sha
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
