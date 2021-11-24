package filesystem

import (
	"errors"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

const defaultReadLimit = 100


func CheckDirPathFormat(path string) bool {
	if len(path) == 0 {
		return false
	}

	if path[0] != rootDir {
		return false
	}

	return true
}


// CheckOrCreateDir check path is a dir, if not exist, create dir.
// If path is a exist dir or create a new dir according path, return nil.
func CheckOrCreateDir(dirPath string) error {
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		err = os.MkdirAll(dirPath, permDirDefault)
		if err != nil {
			return err
		}

		return nil
	}

	if !dirInfo.IsDir() {
		return unix.ENOTDIR
	}

	if dirInfo.Mode().Perm() == permDirDefault {
		return nil
	}

	err = os.Chmod(dirPath, permDirDefault)
	if err != nil {
		return err
	}

	return nil
}
