package files

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

func WatchFiles(files []File) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

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

	if err := watchFiles(files, watcher, "./"); err != nil {
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
