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
	slash            = '/'
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

	isReservedDir := flag.Bool(
		"reserved-dir",
		false,
		"if specify path is dir, reserved dir")

	flag.Parse()

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)
	log.Println("[rmWrapper-Info]New rm request, relativePath:", *relativePath,
		"mountPoint:", *mountPath,
		"isReservedDir:", *isReservedDir)
	log.Println("[rmWrapper-Info]Start check")

	log.Println("[rmWrapper-Info]Start check path format")
	var (
		isPathAvailable bool
		path            string
		err             error
	)

	isPathAvailable = filesystem.CheckDirPathFormat(*mountPath)
	if !isPathAvailable {
		log.Println("[rmWrapper-Error]Unavailable format of mount point:", *mountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *relativePath == emptyValue {
		log.Println("[rmWrapper-Error]Unavailable format of relative path:", *relativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	path, err = filesystem.AbsolutePath(*mountPath, *relativePath)
	if err != nil {
		log.Println("[rmWrapper-Error]Unavailable format of mount point:", *mountPath,
			"or relative path:", *relativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}


	pathLen := len(path)
	rmPath := path
	if rmPath[pathLen-1] == slash {
		rmPath = rmPath[:pathLen-1]
	}

	isPathAvailable = filesystem.CheckFilePathFormat(rmPath)
	if !isPathAvailable {
		log.Println("[rmWrapper-Error]Unavailable format of path:", rmPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[rmWrapper-Info]Check path format...OK")

	var (
		pInfo    os.FileInfo
		retryNum int
	)
	log.Println("[rmWrapper-Info]Start check path is exist")
	for {
		if retryNum >= waitNFSCcliLimit {
			log.Println("[rmWrapper-Info]Path:", rmPath, "is not exist, retry stat num:", retryNum)
			log.Println("[rmWrapper-Info]Check path is exist...NotExist")
			log.Println("[rmWrapper-Info]Path that rm is not exist, exit with 0")
			os.Exit(exit_code.Succeed)
		}

		time.Sleep(waitNFSCliUpdate * time.Second)
		pInfo, err = os.Stat(rmPath)
		if err == nil {
			break
		}

		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[rmWrapper-Error]Failed to stat path:", rmPath, "and err:", err.Error())
			filesystem.Exit(err)
		}

		retryNum += 1
	}
	log.Println("[rmWrapper-Info]Check path is exist...Exist")
	log.Println("[rmWrapper-Info]Start check path mount filesystem")
	err = filesystem.IsMountPath(*mountPath)
	if err != nil {
		log.Println("[rmWrapper-Error]Failed to check path mount filesystem, err:", err.Error())
		filesystem.Exit(err)
	}
	log.Println("[rmWrapper-Info]Check path mount filesystem...OK")

	/*
		path is not exist -> exit with code 0;
		path is exist:
			- path is file -> remove file;
			- path is dir:
				- if reserved dir:
					- remove dir -> create empty dir with same name;
				- if not reserved dir:
					- remove dir;
	 */

	// rm path is file
	if !pInfo.IsDir() {
		log.Println("[rmWrapper-Info]Start remove file:", rmPath)
		err = os.Remove(rmPath)
		if err != nil {
			log.Println("[rmWrapper-Error]Failed to remove file:", rmPath, "and err:", err.Error())
			filesystem.Exit(err)
		}
		log.Println("[rmWrapper-Info]End remove file:", rmPath)
		os.Exit(exit_code.Succeed)
	}

	// rm path is dir
	log.Println("[rmWrapper-Info]Start remove dir:", rmPath, "isReservedDir:", *isReservedDir)
	err = os.RemoveAll(rmPath)
	if err != nil {
		log.Println("[rmWrapper-Error]Failed to remove dir:", rmPath,
			"isReservedDir:", *isReservedDir,"and err:", err.Error())
		filesystem.Exit(err)
	}
	log.Println("[rmWrapper-Info]End remove dir:", rmPath, "isReservedDir:", *isReservedDir)

	if *isReservedDir {
		err = filesystem.CheckOrCreateDir(rmPath)
		if err != nil {
			log.Println("[rmWrapper-Error]Failed to create dir:", rmPath,
				"isReservedDir:", *isReservedDir, "and err:", err.Error())
			filesystem.Exit(err)
		}
		log.Println("[rmWrapper-Info]Succeed to create dir:", rmPath,
			"isReservedDir:", *isReservedDir)
	}

	os.Exit(exit_code.Succeed)
}
