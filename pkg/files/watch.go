package files

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
			md.lastInbound = &fe
			if err := md.ProcessFileEvent(&fe); err != nil {
				glog.Error(err)
			}
		}
	}()
	return
}

type FileEvent struct {
	Type         fsnotify.Op
	Name         string
	PreviousFile []string
	File         *File
	Local        bool
}

func (fe *FileEvent) isUnchanged() bool {
	if fe.PreviousFile == nil && fe.File.Parts == nil {
		return true
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
			glog.Error("Can't find parent", parts)
		}
	}
	return
}

func (md *Metadata) ProcessFileEvent(fe *FileEvent) error {

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

func (md *Metadata) processWatcherEvent(event fsnotify.Event, c *websocket.Conn, watcher *fsnotify.Watcher) {
	if event.Op == fsnotify.Chmod {
		// don't care about chmod at the moment...
		return
	}
	glog.Info("event:", event)
	// if md.lastInbound != nil &&
	// 	md.lastInbound.Name == event.Name &&
	// 	md.lastInbound.Type == event.Op {
	// 	// quite a bit of a hack to prevent duplicate events
	//  // disabling for the moment
	// 	return
	// }
	location := "./" + event.Name

	file, ok := md.fileMap[location]
	if ok != true && event.Op != fsnotify.Create {
		glog.Info(event.Name, "File not found in map")
		return
	}
	if file != nil && event.Op == fsnotify.Create {
		if err := watcher.Add(event.Name); err != nil {
			glog.Error(err)
		}
		// we already have the creation tracked, but it is
		// being created, so add to watcher
		return
	}
	if file == nil && event.Op == fsnotify.Create {
		// assume file is created locally
		// a remote event would have been put in the map
		var err error
		file, err = readFileAndUpload(md.Host, location)
		if err != nil {
			glog.Error(err)
		}
		md.fileMap[location] = file
		if err := watcher.Add(event.Name); err != nil {
			glog.Error(err)
		}
	}
	if file == nil && event.Op == fsnotify.Write {
		glog.Error("write with null tracked file")
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
	if event.Op == fsnotify.Write {
		hashes, err := uploadFile(md.Host, location)
		if err != nil {
			glog.Error(err)
		}
		fe.File.Parts = hashes
		if fe.isUnchanged() {
			return
		}
	}

	if err := md.ProcessFileEvent(&fe); err != nil {
		glog.Error(err)
	}

	fe.Local = false
	bytes, err := json.Marshal(fe)
	if err != nil {
		glog.Error(err)
	}
	if err := c.WriteMessage(websocket.BinaryMessage, bytes); err != nil {
		glog.Error(err)
	}

	// TODO: rename
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
				if strings.HasPrefix(event.Name, "./") {
					event.Name = event.Name[2:]
				}
				md.processWatcherEvent(event, c, watcher)
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
