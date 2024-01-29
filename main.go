package main

import (
	"FileAutoBackup/SimpleLogFormatter"
	"flag"
	"fmt"
	"os"
	fp "path/filepath"
	"strconv"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
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
	c := flag.String("c", "", "配置文件路径, 默认为程序目录下的 config.yaml")
	d := flag.Bool("debug", false, "debug mode~")
	t := flag.Bool("trace", false, "trace mode!")
	flag.Parse()
	if *c != "" {
		f.configPath = *c
	}
	if *d {
		log.SetLevel(log.DebugLevel)
	}
	if *t {
		log.SetLevel(log.TraceLevel)
	}

	if *d && *t {
		log.Panic("y u bully me")
	}

	return f
}

func startAll() {
	for key, session := range config {
		err := watch(
			session.Dir, session.Files, func(event fsnotify.Event) {
				sessionHandler(session, event)
			},
		)
		if err != nil {
			log.Fatal("failed to watch dir ", session.Dir, " in ", key, ": ", err)
		}
	}
}

func sessionHandler(s *Session, event fsnotify.Event) {
	times := strconv.FormatInt(time.Now().Unix(), 10)

	switch len(s.Files) {
	case 0: // 备份整个文件夹
		if time.Since(s.lastBackupTime) < s.MinimumInterval && s.lastBackupTime != launchTime {
			log.Debug(s.Dir, " changed again but less than ", s.MinimumInterval)
			return
		}

		log.Info(s.Dir, " changed again after ", time.Since(s.lastBackupTime))
		s.lastBackupTime = time.Now()

		time.Sleep(time.Second) //等待更新方写入

		if s.Compression {
			destPathInTar := fp.ToSlash(fp.Join(times))
			destTar := fp.ToSlash(fp.Join(s.CopyTo, times+".tar.gz"))
			err := archiveFiles([]string{s.Dir}, destPathInTar, destTar)
			if err != nil {
				log.Error("failed to archive ", s.Dir, " to ", destTar, ": ", err)
			}

		} else {
			destDir := fp.Join(s.CopyTo, times, fp.Base(s.Dir))
			err := copyDir(s.Dir, destDir)
			if err != nil {
				log.Error("failed to copy ", s.Dir, " to ", destDir, ": ", err)
			}
		}

	default: // 备份个别文件
		if time.Since(s.lastBackupTime) < s.MinimumInterval && s.lastBackupTime != launchTime {
			log.Debug(event.Name, " changed again but less than ", s.MinimumInterval)
			return
		}

		log.Info(s.Dir, " changed again after ", time.Since(s.lastBackupTime))
		s.lastBackupTime = time.Now()

		time.Sleep(time.Second) //等待更新方写入

		if s.Compression {
			destPathInTar := fp.ToSlash(fp.Join(times))
			destTar := fp.ToSlash(fp.Join(s.CopyTo, times+".tar.gz"))
			err := archiveFiles([]string{event.Name}, destPathInTar, destTar)
			if err != nil {
				log.Error("failed to archive ", event.Name, " to ", destTar, ": ", err)
			}

		} else {
			destFile := fp.Join(s.CopyTo, times, fp.Base(event.Name))
			err := copyFile(event.Name, destFile)
			if err != nil {
				log.Error("failed to copy ", event.Name, " to ", destFile, ": ", err)
			}
		}

	}
}

func safeCreate(name string) (*os.File, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		err := os.MkdirAll(fp.Dir(name), os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("failed to mkdir: %w", err)
		}
	}

	return os.Create(name)
}
