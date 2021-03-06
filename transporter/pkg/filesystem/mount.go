package filesystem

import (
	"errors"
	"io/fs"
	"strings"

	"golang.org/x/sys/unix"
)

// Available mount filesystem
const (
	NFS     = 0x6969
	LUSTRE0 = 0x0BD00BD0
	LUSTRE1 = 0x0BD00BD1
)

var fsList = []int64{
	NFS,
	LUSTRE0,
	LUSTRE1,
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
	var err error
	fsInfo := unix.Statfs_t{}

	for {
		err = unix.Statfs(path, &fsInfo)
		if err == nil {
			break
		}

		// We have to check EINTR here, per issues 11180 and 39237.
		if err == unix.EINTR {
			continue
		}

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

func AbsolutePath(mountPoint, relativePath string) (string, error) {
	isAvailable := CheckDirPathFormat(mountPoint)
	if !isAvailable {
		return "", errors.New("unavailable format of mount point")
	}

	if mountPoint[len(mountPoint)-1] != slash {
		mountPoint += slashStr
	}

	if len(relativePath) == 0 {
		return mountPoint, nil
	}

	if relativePath == slashStr {
		return mountPoint, nil
	}

	if relativePath[0] == slash {
		relativePath = relativePath[1:]
	}

	return mountPoint + relativePath, nil
}
