package files

import (
	"fmt"
	"log"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

func (md *Metadata) connectToServer() (c *websocket.Conn, err error) {
	c, _, err = websocket.DefaultDialer.Dial(
		fmt.Sprintf("%s/events/%s",
			strings.Replace(md.Host, "http", "ws", 1),
			md.Name,
		),
		nil,
	)
	if err != nil {
		return
	}
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				glog.Fatal("ws read error:", err)
				return
			}
			log.Printf("recv: %s\n", message)
		}
	}()
	return
}

func (md *Metadata) WatchFiles() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	c, err := md.connectToServer()
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	c.WriteMessage(websocket.TextMessage, []byte("hello"))

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	if err := watchFiles(md.Files, watcher, "./"); err != nil {
		return err
	}
	<-done
	return nil
}

func watchFiles(files []File, watcher *fsnotify.Watcher, path string) error {
	if err := watcher.Add(path); err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir {
			if err := watchFiles(file.Contents, watcher, path+file.Name+"/"); err != nil {
				return err
			}
		} else {
			if err := watcher.Add(path + file.Name); err != nil {
				return err
			}
		}
	}
	return nil
}
