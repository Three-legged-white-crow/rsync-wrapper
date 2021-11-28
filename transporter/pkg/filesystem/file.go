package filesystem

import (
	"errors"
	"io/fs"
	"os"
	"strings"

	"golang.org/x/sys/unix"
	"transporter/pkg/exit_code"
)

const (
	slash           = '/'
	star            = '*'
	slashStr        = "/"
	permDirDefault  = 0775
	permFileDefault = 0775
)


func Exit(err error) {
	if errors.Is(err, fs.ErrPermission) {
		os.Exit(exit_code.ErrPermissionDenied)
	}

	if errors.Is(err, fs.ErrNotExist) {
		os.Exit(exit_code.ErrNoSuchFileOrDir)
	}

	if errors.Is(err, fs.ErrExist) {
		os.Exit(exit_code.ErrFileIsExists)
	}

	if errors.Is(err, ErrUnavailableFileSystem) {
		os.Exit(exit_code.ErrUnknownFSType)
	}

	os.Exit(exit_code.ErrSystem)
}


func CheckFilePathFormat(path string) bool {

	l := len(path)

	if l == 0 {
		return false
	}

	if path[0] != slash {
		return false
	}

	lastChar := path[l-1]

	if lastChar == slash {
		return false
	}

	if lastChar == star && len(path) > 1 {
		if path[l-2] == slash {
			return false
		}
	}

	return true
}


func CheckOrCreateFile(filePath string, isOverWrite bool) error {
	fInfo, err := os.Stat(filePath)
	if err == nil {
		if fInfo.Mode().Perm() == permFileDefault {
			return nil
		}

		err = os.Chmod(filePath, permFileDefault)
		if err != nil {
			return err
		}

		return nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	parentDirIndex := strings.LastIndex(filePath, slashStr)
	if parentDirIndex > 1 {
		parentDirPath := filePath[:parentDirIndex]
		err = os.MkdirAll(parentDirPath, permDirDefault)
		if err != nil {
			return err
		}
	}

	var openFlag int = unix.O_RDWR|unix.O_CREAT
	if isOverWrite {
		openFlag = openFlag|unix.O_TRUNC
	}
	_, err = os.OpenFile(filePath, openFlag, permFileDefault)
	if err != nil {
		return err
	}

	return nil
}