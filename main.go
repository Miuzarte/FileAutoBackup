package main

import (
	"FileAutoBackup/SimpleLogFormatter"
	"flag"
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"
)

type flags struct {
	configPath string
}

var (
	launchTime = time.Now()
	config     = make(Config)
)

func main() {
	initLogger()
	flags := initFlag()
	err := initConfig(flags.configPath)
	if err != nil {
		log.Fatal(err)
	}
	startAll()
	log.Info("Listening...")
	<-make(chan struct{})
}

func initLogger() {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(
		&SimpleLogFormatter.LogFormat{},
	)
}

func initFlag() (f flags) {
	f = flags{}
	c := flag.String("c", "", "配置文件路径, 默认为程序目录下的config.yaml")
	flag.Parse()
	if *c != "" {
		f.configPath = *c
	}
	return f
}

func startAll() {
	for key, session := range config {
		if session.MinimumInterval <= time.Second {
			session.MinimumInterval = time.Second
		}
		session.lastBackupTime = launchTime

		err := watch(
			session.Dir, session.Files, func(event fsnotify.Event) {
				switch len(session.Files) {
				case 0:
					if time.Since(session.lastBackupTime) < session.MinimumInterval && session.lastBackupTime != launchTime {
						log.Info(session.Dir, " changed again but less than ", session.MinimumInterval)
					} else {
						log.Info(session.Dir, " changed again after ", time.Since(session.lastBackupTime))
						session.lastBackupTime = time.Now()

						destDir := filepath.Join(
							session.CopyTo, strconv.FormatInt(time.Now().Unix(), 10), filepath.Base(session.Dir),
						)
						err := copyDir(session.Dir, destDir)
						if err != nil {
							log.Fatal("failed to copy ", session.Dir, " to ", destDir, ": ", err)
						}
					}
				default:
					if time.Since(session.lastBackupTime) < session.MinimumInterval && session.lastBackupTime != launchTime {
						log.Info(event.Name, " changed again but less than ", session.MinimumInterval)
					} else {
						log.Info(session.Dir, " changed again after ", time.Since(session.lastBackupTime))
						session.lastBackupTime = time.Now()

						destFile := filepath.Join(
							session.CopyTo, strconv.FormatInt(time.Now().Unix(), 10), filepath.Base(event.Name),
						)
						err := copyFile(event.Name, destFile)
						if err != nil {
							log.Fatal("failed to copy ", event.Name, " to ", destFile, ": ", err)
						}
					}
				}
			},
		)
		if err != nil {
			log.Fatal("failed to watch dir ", session.Dir, " in ", key, ": ", err)
		}
	}
}

func copyFile(src, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destDir, _ := filepath.Split(dest)
	_, err = os.Stat(destDir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(destDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	destinationFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

func copyDir(src, dest string) error {
	err := os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return err
	}

	return filepath.Walk(
		src, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			destinationPath := filepath.Join(dest, path[len(src):])

			if info.IsDir() {
				return os.MkdirAll(destinationPath, os.ModePerm)
			} else {
				return copyFile(path, destinationPath)
			}
		},
	)
}

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
					_, thisName := filepath.Split(event.Name)
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
