package filesystem

import (
	"errors"
	"io/fs"
	"strings"

	"golang.org/x/sys/unix"
)

// Available mount filesystem
const (
	NFS       = 0x6969
	LUSTRE    = 0x0BD00BD0
)

var fsList = []int64{
	NFS,
	LUSTRE,
}

var ErrUnavailableFileSystem = errors.New("unavailable filesystem")

func IsAvailableFileSystem(fsType int64) bool {
	for _, avfstype := range fsList {
		if fsType == avfstype {
			return true
		}
	}

	return false
}

func IsMountPath(path string) error {

	fsInfo := unix.Statfs_t{}
	err := unix.Statfs(path, &fsInfo)
	if err != nil {
		return err
	}

	isAvailableFS := IsAvailableFileSystem(fsInfo.Type)
	if !isAvailableFS {
		return ErrUnavailableFileSystem
	}

	return nil
}


func IsMountPathList(pathList ...string) error {
	var err error

	for _, path := range pathList {
		err = IsMountPath(path)
		if err != nil {
			return err
		}
	}

	return nil
}

// CheckMountAllPath check is path is a mount point and is mount filesystem available.
// Because some of the directories may not exist,
// it will check step by step to the parent directory
// until the subdirectories under the root directory.
func CheckMountAllPath(path string) error {
	numSlash := strings.Count(path, slashStr)
	var (
		indexLastSlash int
		pathMountCheck string
		err            error
	)
	pathMountCheck = path
	for i := 0; i < numSlash-1; i++ {
		indexLastSlash = strings.LastIndex(pathMountCheck, slashStr)
		pathMountCheck = pathMountCheck[:indexLastSlash]
		err = IsMountPath(pathMountCheck)
		if err == nil {
			return nil
		}

		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}

	return err
}