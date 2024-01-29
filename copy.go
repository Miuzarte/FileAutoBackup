package main

import (
	"io"
	"os"
	fp "path/filepath"
)

func copyDir(src, dest string) error {
	err := os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return err
	}

	return fp.Walk(
		src, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			destinationPath := fp.Join(dest, path[len(src):])

			if info.IsDir() {
				return os.MkdirAll(destinationPath, os.ModePerm)
			} else {
				return copyFile(path, destinationPath)
			}
		},
	)
}

func copyFile(src, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destDir, _ := fp.Split(dest)
	_, err = os.Stat(destDir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(destDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	destinationFile, err := safeCreate(dest)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}
