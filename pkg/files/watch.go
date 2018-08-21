package files

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
			var fe FileEvent
			if err := json.Unmarshal(message, &fe); err != nil {
				glog.Error(err)
			}
			if err := md.ProcessFileEvent(&fe); err != nil {
				glog.Error(err)
			}
		}
	}()
	return
}

type FileEvent struct {
	Type         fsnotify.Op //Remove/Write/Create
	Name         string
	PreviousFile []string
	File         *File
	Local        bool
}

func (fe *FileEvent) isUnchanged() bool {
	if fe.PreviousFile == nil {
		return false
	}
	if len(fe.PreviousFile) != len(fe.File.Parts) {
		return false
	}
	for i := range fe.PreviousFile {
		if fe.PreviousFile[i] != fe.File.Parts[i] {
			return false
		}
	}
	return true
}

func (md *Metadata) findParent(eventName string) (isRoot bool, parent *File) {
	name := "./" + eventName
	parts := strings.Split(name, "/")
	if len(parts) == 2 {
		isRoot = true
	} else {
		var ok bool
		parent, ok = md.fileMap[strings.Join(parts[1:], "/")]
		if ok != true {
			glog.Error("Can't find parent")
		}
	}
	return
}

func (md *Metadata) ProcessFileEvent(fe *FileEvent) error {
	var file *File
	var err error
	if fe.Local &&
		fe.Type&fsnotify.Remove != fsnotify.Remove &&
		fe.Type&fsnotify.Chmod != fsnotify.Chmod {
		file, err = readFileAndUpload(md.Host, "./"+fe.Name)

		if err != nil {
			return err
		}
		if fe.File != nil {
			fe.File.Parts = file.Parts
			fe.File.Mode = file.Mode
			fe.File.ModTime = file.ModTime
		}
	}

	if fe.Type&fsnotify.Remove == fsnotify.Remove {
		glog.Info("remove file:", fe.Name)
		// isRoot, parent := md.findParent(fe.Name)
		// TODO: handle removals
		// what happens with directories?
	}
	if fe.Type&fsnotify.Write == fsnotify.Write {
		if !fe.Local {
			glog.Info("non local write event")
			var buf bytes.Buffer
			if err := writeFileBody(md.Host, &buf, fe.File); err != nil {
				glog.Error(err)
			}
			ioutil.WriteFile("./"+fe.Name, buf.Bytes(), fe.File.Mode)
		}
	}
	if fe.Type&fsnotify.Create == fsnotify.Create {
		if !fe.Local {
			err := _createFile(md.Host, "./"+fe.Name, fe.File)
			if err != nil {
				glog.Error(err)
			}
		} else {
			fe.File = file
		}
		// add new file object
		isRoot, parent := md.findParent(fe.Name)
		if isRoot {
			md.Files = append(md.Files, fe.File)
		} else if parent != nil {
			parent.Contents = append(parent.Contents, fe.File)
		}
		md.fileMap["./"+fe.Name] = fe.File
	}

	// update tree
	return nil
}

func (md *Metadata) processWatcherEvent(event fsnotify.FileEvent, c *websocket.Conn) {
	file, ok := md.fileMap["./"+event.Name]
	if ok != true && event.Op&fsnotify.Create != fsnotify.Create {
		glog.Info(event.Name, "File not found in map")
		continue
	}
	fe := FileEvent{
		Type:  event.Op,
		Name:  event.Name,
		File:  file,
		Local: true,
	}
	if file != nil {
		fe.PreviousFile = file.Parts
	}
	err := md.ProcessFileEvent(&fe)
	if err != nil {
		glog.Error(err)
	}
	if event.Op&fsnotify.Write == fsnotify.Write {
		log.Println("modified file:", event.Name)
		if fe.isUnchanged() {
			continue
		}
	}

	fe.Local = false
	bytes, err := json.Marshal(fe)
	if err != nil {
		glog.Error(err)
	}
	if err := c.WriteMessage(websocket.BinaryMessage, bytes); err != nil {
		glog.Error(err)
	}

	// rename
	if event.Op&fsnotify.Remove == fsnotify.Remove {
		glog.Info("remove file:", event.Name)
		// what happens with directories?
	}

	if event.Op&fsnotify.Create == fsnotify.Create {
		// add new file object
		glog.Info("created file:", event.Name)
		if err := watcher.Add(event.Name); err != nil {
			glog.Error(err)
		}
		// TOOD: get parent
		// TODO: add file
		// md.fileMap["./"+event.Name] =
	}
}

func (md *Metadata) WatchFiles() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		glog.Fatal(err)
	}
	defer watcher.Close()

	c, err := md.connectToServer()
	if err != nil {
		glog.Fatal(err)
	}
	defer c.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				glog.Info("event:", event)
				md.processWatcherEvent(event, c)
			case err := <-watcher.Errors:
				glog.Info("error:", err)
			}
		}
	}()

	if err := md.watchFiles(md.Files, watcher, "./"); err != nil {
		return err
	}
	<-done
	return nil
}

func (md *Metadata) watchFiles(files []*File, watcher *fsnotify.Watcher, path string) error {
	if err := watcher.Add(path); err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir {
			if err := md.watchFiles(file.Contents, watcher, path+file.Name+"/"); err != nil {
				return err
			}
			md.fileMap[path+file.Name] = file
		} else {
			if err := watcher.Add(path + file.Name); err != nil {
				return err
			}
			md.fileMap[path+file.Name] = file
		}
	}
	return nil
}
