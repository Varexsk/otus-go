package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/cheggaaa/pb/v3"
)

var (
	ErrUnsupportedFile       = errors.New("unsupported file")
	ErrNotExist              = errors.New("file not exist")
	ErrOffsetExceedsFileSize = errors.New("offset exceeds file size")
)

func Copy(fromPath, toPath string, offset, limit int64) error {
	sourceFile, err := os.OpenFile(fromPath, os.O_RDONLY, 0)
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNotExist
	} else if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer sourceFile.Close()

	fileInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("source stat: %w", err)
	}
	if !fileInfo.Mode().IsRegular() {
		return ErrUnsupportedFile
	}
	if offset > fileInfo.Size() {
		return ErrOffsetExceedsFileSize
	}

	toCopy := fileInfo.Size() - offset
	if limit > 0 && limit < toCopy {
		toCopy = limit
	}

	if _, err = sourceFile.Seek(offset, io.SeekStart); err != nil {
		return fmt.Errorf("seek: %w", err)
	}

	destFile, err := os.Create(toPath)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer destFile.Close()

	bar := pb.Full.Start64(toCopy)
	defer bar.Finish()
	barWriter := bar.NewProxyWriter(destFile)

	_, err = io.CopyN(barWriter, sourceFile, toCopy)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
