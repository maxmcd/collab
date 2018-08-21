package main

import (
	"math/rand"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var UPGRADER = websocket.Upgrader{}
var LISTENERS = make(map[string]map[string]Listener)

type Listener struct {
	channel chan []byte
	key     string
	host    bool
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func genKey() string {
	n := 10
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func newListener(name string) Listener {
	var host bool
	if _, ok := LISTENERS[name]; !ok {
		LISTENERS[name] = make(map[string]Listener)
		host = true
	}
	listener := Listener{
		channel: make(chan []byte, 2),
		key:     genKey(),
		host:    host,
	}
	LISTENERS[name][listener.key] = listener
	return listener
}

func WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	c, err := UPGRADER.Upgrade(w, r, nil)
	if err != nil {
		glog.Info("upgrade:", err)
		return
	}
	defer c.Close()

	listener := newListener(name)

	go func() {
		for {
			message := <-listener.channel
			err := c.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				glog.Error(err)
			}
		}
	}()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			glog.Info("read error: ", err)
			break
		}
		glog.Info("recv: ", string(message))
		// err = c.WriteMessage(mt, message)
		// if err != nil {
		// 	log.Println("write:", err)
		// 	break
		// }
		for k, v := range LISTENERS[name] {
			if k != listener.key {
				v.channel <- message
			}
		}
	}
	if listener.host {
		// Free directory lock
		delete(DIRECTORIES, name)
	}
	delete(LISTENERS[name], listener.key)
}
