package main

import (
	"errors"
	"flag"
	"io/fs"
	"log"
	"os"
	"time"

	"transporter/pkg/exit_code"
	"transporter/pkg/filesystem"
)

const (
	emptyValue       = "empty"
	typeFile         = "file"
	typeDir          = "dir"
	typeAll          = "all"
	waitNFSCliUpdate = 5
	waitNFSCcliLimit = 5
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

	typeStat := flag.String(
		"type",
		typeAll,
		"stat type: file,dir,all")

	flag.Parse()

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)
	log.Println("[statWrapper-Info]New stat request, relativePath:", *relativePath,
		"mountPath:", *mountPath,
		"type:", *typeStat)
	log.Println("[statWrapper-Info]Start check")

	log.Println("[statWrapper-Info]Start check path format")
	var (
		isPathAvailable bool
		path            string
		err             error
	)

	isPathAvailable = filesystem.CheckDirPathFormat(*mountPath)
	if !isPathAvailable {
		log.Println("[statWrapper-Error]Unavailable format of mount point:", *mountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *relativePath == emptyValue {
		log.Println("[statWrapper-Error]Unavailable format of relative path:", *relativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	path, err = filesystem.AbsolutePath(*mountPath, *relativePath)
	if err != nil {
		log.Println("[statWrapper-Error]Unavailable format of mount point:", *mountPath,
			"or relative path:", *relativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	switch *typeStat {
	case typeFile:
		isPathAvailable = filesystem.CheckFilePathFormat(path)
	case typeDir, typeAll:
		isPathAvailable = filesystem.CheckDirPathFormat(path)

	default:
		log.Println("[statWrapper-Error]Unsupport stat type:", *typeStat, "with path:", path)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if !isPathAvailable {
		log.Println("[statWrapper-Error]Unavailable format of path:", path, "stat type:", *typeStat)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[statWrapper-Info]Check path format...OK")
	log.Println("[statWrapper-Info]Start check mount filesystem")
	err = filesystem.IsMountPath(*mountPath)
	if err != nil {
		log.Println("[statWrapper-Error]Failed to check mount filesystem, err:", err.Error())
		filesystem.Exit(err)
	}
	log.Println("[statWrapper-Info]Check mount filesystem...OK")
	log.Println("[statWrapper-Info]End check")

	var (
		pInfo    os.FileInfo
		retryNum int
	)
	log.Println("[statWrapper-Info]Start stat path:", path)
	for {
		if retryNum >= waitNFSCcliLimit {
			log.Println("[statWrapper-Info]Stat path:", path,
				"is not exist, retry stat num:", retryNum)
			log.Println("[statWrapper-Info]Path that stat is not exist, exit with 2 (No such file or directory)")
			os.Exit(exit_code.ErrNoSuchFileOrDir)
		}

		time.Sleep(waitNFSCliUpdate * time.Second)
		pInfo, err = os.Stat(path)
		if err == nil {
			isPathDir := pInfo.IsDir()
			switch *typeStat {
			case typeFile:
				if isPathDir {
					log.Println("[statWrapper-Error]Stat file:", path,
						"is exist but is dir, retry stat num:", retryNum)
					log.Println("[statWrapper-Error]Path that stat is exist but is dir, exit with 21")
					os.Exit(exit_code.ErrIsDirectory)
				}
				log.Println("[statWrapper-Info]Stat file:", path,
					"is exist and is file, retry stat num:", retryNum)
				log.Println("[statWrapper-Info]Path that stat is exist and is file, exit with 0")
				os.Exit(exit_code.Succeed)

			case typeDir:
				if isPathDir {
					log.Println("[statWrapper-Info]Stat dir:", path,
						"is exist and is dir, retry stat num:", retryNum)
					log.Println("[statWrapper-Info]Path that stat is exist and is dir, exit with 0")
					os.Exit(exit_code.Succeed)
				}
				log.Println("[statWrapper-Error]Stat dir:", path,
					"is exist but is file, retry stat num:", retryNum)
				log.Println("[statWrapper-Error]Path that stat is exist but is file, exit with 20")
				os.Exit(exit_code.ErrNotDirectory)

			default:
				log.Println("[statWrapper-Info]Stat path:", path,
					"is exist, retry stat num:", retryNum)
				log.Println("[statWrapper-Info]Path that stat is exist, exit with 0")
				os.Exit(exit_code.Succeed)
			}

		}

		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[statWrapper-Error]Failed to stat path:", path, "and err:", err.Error())
			filesystem.Exit(err)
		}

		retryNum += 1
	}
}
