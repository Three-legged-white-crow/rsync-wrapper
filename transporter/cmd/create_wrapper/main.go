package main

import (
	"errors"
	"flag"
	"io/fs"
	"log"
	"os"
	"strings"

	"transporter/pkg/exit_code"
	"transporter/pkg/filesystem"
)

const (
	typeFile      = "file"
	typeDir       = "dir"
	slash         = "/"
	emptyValue    = "empty"
)

func main() {

	path := flag.String("path", emptyValue, "path that wait create")
	typeCreate := flag.String("type", emptyValue, "available create types: file,dir")
	flag.Parse()

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)
	log.Println("[createWrapper-Info]New create request, path:", *path, "type:", *typeCreate)
	log.Println("[createWrapper-Info]Start check")

	var isPathAvailable bool
	switch *typeCreate {
	case typeFile:
		isPathAvailable = filesystem.CheckFilePathFormat(*path)
	case typeDir:
		isPathAvailable = filesystem.CheckDirPathFormat(*path)
	default:
		log.Println("[createWrapper-Error]Unsupport create type:", *typeCreate)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if !isPathAvailable {
		log.Println("[createWrapper-Error]Unavailable path:", *path)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[createWrapper-Info]Check path format...OK")

	err := checkMount(*path)
	if err != nil {
		log.Println("[createWrapper-Error]Failed to check mount filesystem, err:", err.Error())
		filesystem.Exit(err)
	}
	log.Println("[createWrapper-Info]Check mount filesystem...OK")
	log.Println("[createWrapper-Info]End check")

	log.Println("[createWrapper-Info]Start create")
	switch *typeCreate {
	case typeDir:
		err = filesystem.CheckOrCreateDir(*path)

	case typeFile:
		err = filesystem.CheckOrCreateFile(*path)

	default:
		log.Println("[createWrapper-Error]Unsupport create type:", *typeCreate)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if err != nil {
		log.Println("[createWrapper-Error]Failed to create path:", *path, "and err:", err.Error())
		filesystem.Exit(err)
	}

	log.Println("[createWrapper-Info]End create")
	log.Println("[createWrapper-Info]Succeed to create path:", *path, "type:", *typeCreate)
	os.Exit(exit_code.Succeed)
}

// checkMount check is path is a mount point and is mount filesystem available.
// Because some of the directories may not exist,
// it will check step by step to the parent directory
// until the subdirectories under the root directory.
func checkMount(path string) error {
	numSlash := strings.Count(path, slash)
	var (
		indexLastSlash int
		pathMountCheck string
		err            error
	)
	pathMountCheck = path
	for i := 0; i < numSlash-1; i++ {
		indexLastSlash = strings.LastIndex(pathMountCheck, slash)
		pathMountCheck = pathMountCheck[:indexLastSlash]
		err = filesystem.IsMountPath(pathMountCheck)
		if err == nil {
			return nil
		}

		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}

	return err
}