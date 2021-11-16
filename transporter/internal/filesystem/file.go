package filesystem

import (
	"errors"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

const permDir = 0775

// CheckOrCreateDir check path is a dir, if not exist, create dir.
// If path is a exist dir or create a new dir according path, return nil.
func CheckOrCreateDir(dirPath string) error {
	f, err := os.Open(dirPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		err = os.MkdirAll(dirPath, permDir)
		if err != nil {
			return err
		}

		return nil
	}

	fInfo, err := f.Stat()
	if err != nil {
		return err
	}

	if !fInfo.IsDir() {
		return unix.ENOTDIR
	}

	return nil
}
