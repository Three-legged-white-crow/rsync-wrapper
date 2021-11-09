package filesystem

import (
	"errors"

	"golang.org/x/sys/unix"
)

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
