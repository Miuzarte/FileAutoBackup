package main

import (
	"fmt"
	fp "path/filepath"
	"slices"
	"sync"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

func watch(dir string, names []string, f func(fsnotify.Event)) (err error) {
	if f == nil {
		return fmt.Errorf("nil callback error")
	}
	initErr := make(chan error)
	go func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			initErr <- err
			return
		}
		err = watcher.Add(dir)
		if err != nil {
			initErr <- err
			return
		}
		initErr <- nil
		defer func(watcher *fsnotify.Watcher) {
			if err := watcher.Close(); err != nil {
				log.Error("failed to close watcher: ", err)
			}
		}(watcher)

		eventWG := sync.WaitGroup{}
		eventWG.Add(1)
		go func() {
			defer eventWG.Done()
			for {
				select {
				case event, ok := <-watcher.Events:
					log.Trace("File: ", event.Name, ", Op: ", event.Op, ", Ok: ", ok)
					if !ok {
						return
					}
					_, thisName := fp.Split(event.Name)
					if slices.Contains(names, thisName) ||
						len(names) == 0 { //监听整个文件夹的情况
						f(event)
					}

				case err, ok := <-watcher.Errors:
					log.Error(err, ", Ok: ", ok)
					return
				}
			}
		}()
		eventWG.Wait()
	}()
	err = <-initErr
	if err != nil {
		return err
	}
	return nil
}
