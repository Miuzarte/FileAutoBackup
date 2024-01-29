package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	fp "path/filepath"

	log "github.com/sirupsen/logrus"
)

// archiveFiles 将文件打包到 destTar, 在其中以 destPathInTar 路径储存
// srcFiles 必须是绝对路径
func archiveFiles(srcFiles []string, destPathInTar, destTar string) error {
	// 创建或打开目标 tar.gz 文件
	targzW, err := safeCreate(destTar)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", destTar, err)
	}
	defer targzW.Close()

	// 创建 gzip.Writer
	gzipW := gzip.NewWriter(targzW)
	defer gzipW.Close()

	// 创建 tar.Writer
	tarW := tar.NewWriter(gzipW)
	defer tarW.Close()

	// 遍历源文件列表，将每个文件存储到 tar 包中
	for _, src := range srcFiles {
		fileName := fp.Base(src) // 获取文件名
		err := storeToTar(tarW, src, fp.ToSlash(fp.Join(destPathInTar, fileName)))
		if err != nil {
			return fmt.Errorf("failed to store file: %w", err)
		}
	}
	return nil
}

// storeToTar 将文件或目录存储到 tar 包中
func storeToTar(tarW *tar.Writer, src, dest string) error {
	// 获取源文件或目录的信息
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to specify %s: %w", src, err)
	}

	// 如果是目录，则递归遍历目录下的文件并存储到 tar 包中
	if info.IsDir() {
		return fp.Walk(src, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("failed to walk %s: %w", src, err)
			}

			if !info.IsDir() {
				// 计算相对路径
				relPath, err := fp.Rel(src, path)
				if err != nil {
					return fmt.Errorf("failed to calculate relative path: %w", err)
				}

				// 拼接存储路径
				storePath := fp.ToSlash(fp.Join(dest, relPath))

				// 递归调用 storeToTar 存储文件
				return storeToTar(tarW, path, storePath)
			}

			return nil
		})
	}

	// 如果是文件，打开文件
	srcFile, err := os.Open(src)
	defer func() {
		if err := srcFile.Close(); err != nil {
			log.Error(fmt.Errorf("failed to close %s: %w", src, err))
		}
	}()

	if err != nil {
		return fmt.Errorf("failed to open %s", src)
	}

	// 创建 tar 包头信息
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("failed to create tar header: %w", err)
	}

	header.Name = dest // 设置 tar 包头中的文件名
	err = tarW.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}

	// 将文件内容复制到 tar 包中
	_, err = io.Copy(tarW, srcFile)
	if err != nil {
		return err
	}
	return nil
}
