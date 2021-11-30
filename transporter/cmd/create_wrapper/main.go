package main

import (
	"flag"
	"log"
	"os"

	"transporter/pkg/exit_code"
	"transporter/pkg/filesystem"
)

const (
	typeFile      = "file"
	typeDir       = "dir"
	emptyValue    = "empty"
)

func main() {

	mountPath := flag.String(
		"mount-path",
		emptyValue,
		"mount point path")

	relativePath := flag.String(
		"relative-path",
		emptyValue,
		"path relative mount point")

	typeCreate := flag.String(
		"type",
		emptyValue,
		"available create types: file,dir")

	isOverWrite := flag.Bool(
		"overwrite",
		false,
		"if create type is file, truncat exist file")

	isDebug := flag.Bool(
		"debug",
		false,
		"enable debug mode")

	flag.Parse()

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)
	log.Println("[createWrapper-Info]New create request, relative path:", *relativePath,
		"mount point:", *mountPath,
		"type:", *typeCreate,
		"isOverWrite:", *isOverWrite,
		"isDebug:", *isDebug,
	)
	log.Println("[createWrapper-Info]Start check")

	log.Println("[createWrapper-Info]Start check path format")
	var (
		isPathAvailable bool
		path            string
		err             error
		exitCode        int
	)

	isPathAvailable = filesystem.CheckDirPathFormat(*mountPath)
	if !isPathAvailable {
		log.Println("[createWrapper-Error]Unavailable format of mount point:", *mountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *relativePath == emptyValue {
		log.Println("[createWrapper-Error]Unavailable format of relative path:", *relativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	path, err = filesystem.AbsolutePath(*mountPath, *relativePath)
	if err != nil {
		log.Println("[createWrapper-Error]Unavailable format of mount point:", *mountPath,
			"or relative path:", *relativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	switch *typeCreate {
	case typeFile:
		isPathAvailable = filesystem.CheckFilePathFormat(path)
	case typeDir:
		isPathAvailable = filesystem.CheckDirPathFormat(path)
	default:
		log.Println("[createWrapper-Error]Unsupport create type:", *typeCreate)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if !isPathAvailable {
		log.Println("[createWrapper-Error]Unavailable format of create path:", path)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[createWrapper-Info]Check path format...OK")

	if !(*isDebug) {
		log.Println("[createWrapper-Info]Start check mount filesystem")
		err = filesystem.IsMountPath(*mountPath)
		if err != nil {
			log.Println("[createWrapper-Error]Failed to check mount filesystem, err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
		log.Println("[createWrapper-Info]Check mount filesystem...OK")
	}

	log.Println("[createWrapper-Info]End check")

	log.Println("[createWrapper-Info]Start create")
	switch *typeCreate {
	case typeDir:
		err = filesystem.CheckOrCreateDir(path)

	case typeFile:
		err = filesystem.CheckOrCreateFile(path, *isOverWrite)

	default:
		log.Println("[createWrapper-Error]Unsupport create type:", *typeCreate)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if err != nil {
		log.Println("[createWrapper-Error]Failed to create path:", path, "and err:", err.Error())
		exitCode = exit_code.ExitCodeConvertWithErr(err)
		os.Exit(exitCode)
	}

	log.Println("[createWrapper-Info]End create")
	log.Println("[createWrapper-Info]Succeed to create path:", path, "type:", *typeCreate)
	os.Exit(exit_code.Succeed)
}