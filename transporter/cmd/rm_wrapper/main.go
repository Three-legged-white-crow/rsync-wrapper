//go:build amd64 && linux
// +build amd64,linux

package main

import (
	"errors"
	"flag"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	"transporter/pkg/exit_code"
	"transporter/pkg/filesystem"
)

const (
	emptyValue        = "empty"
	slash             = '/'
	slashStr          = "/"
	waitNFSCliUpdate  = 5
	waitNFSCcliLimit  = 5
	PathSeparator     = '/' // OS-specific path separator
	PathListSeparator = ':' // OS-specific path list separator
	reqSize           = 1024
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

	fileSuffix := flag.String(
		"suffix",
		emptyValue,
		"suffix of file to remove")

	isDebug := flag.Bool(
		"debug",
		false,
		"enable debug mode")

	flag.Parse()

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)
	log.Println("[rmWrapper-Info]New rm request, relativePath:", *relativePath,
		"mountPoint:", *mountPath,
		"isReservedDir:", *isReservedDir,
		"isDebug:", *isDebug,
	)
	log.Println("[rmWrapper-Info]Start check")

	log.Println("[rmWrapper-Info]Start check path format")
	var (
		isPathAvailable bool
		path            string
		err             error
		exitCode        int
		isSuffixEmpty   bool
		suffixList      []string
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

	if *fileSuffix == emptyValue {
		isSuffixEmpty = true
		log.Println("[rmWrapper-Info]Not specify file suffix")
	} else {
		suffixList = strings.Split(*fileSuffix, slashStr)
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

	if endsWithDot(rmPath) {
		log.Println("[rmWrapper-Error]Path end with dot:", rmPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	log.Println("[rmWrapper-Info]Check path format...OK")

	if !(*isDebug) {
		log.Println("[rmWrapper-Info]Start check path mount filesystem")
		err = filesystem.IsMountPath(*mountPath)
		if err != nil {
			log.Println("[rmWrapper-Error]Failed to check path mount filesystem, err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
		log.Println("[rmWrapper-Info]Check path mount filesystem...OK")
	}

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

		pInfo, err = os.Stat(rmPath)
		if err == nil {
			break
		}

		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[rmWrapper-Error]Failed to stat path:", rmPath, "and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		time.Sleep(waitNFSCliUpdate * time.Second)
		retryNum += 1
	}
	log.Println("[rmWrapper-Info]Check path is exist...Exist")

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
		if err == nil {
			log.Println("[rmWrapper-Info]End remove file:", rmPath)
			os.Exit(exit_code.Succeed)
		}

		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[rmWrapper-Error]Failed to remove file:", rmPath, "and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		log.Println("[rmWrapper-Info]File is not exist, return succeed directly")
		os.Exit(exit_code.Succeed)
	}

	// rm path is dir
	log.Println("[rmWrapper-Info]Start remove dir:", rmPath,
		"isReservedDir:", *isReservedDir,
		"fileSuffix:", *fileSuffix)
	if !(*isReservedDir) {
		err = os.RemoveAll(rmPath)
		if err != nil {
			log.Println("[rmWrapper-Error]Failed to remove dir:", rmPath,
				"isReservedDir:", *isReservedDir,
				"fileSuffix:", *fileSuffix,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		log.Println("[rmWrapper-Info]End remove dir:", rmPath,
			"isReservedDir:", *isReservedDir,
			"fileSuffix:", *fileSuffix)
		os.Exit(exit_code.Succeed)
	}

	// reserved dir
	err = removeChild(rmPath, isSuffixEmpty, suffixList)
	if err == nil {
		os.Exit(exit_code.Succeed)
	}

	exitCode = exit_code.ExitCodeConvertWithErr(err)
}

func isNeedRemove(fileName string, fileSuffixList []string) bool {
	if fileSuffixList == nil {
		return false
	}

	var isMatch bool
	for _, suffix := range fileSuffixList {
		isMatch, _ = filepath.Match(suffix, fileName)
		if isMatch {
			return true
		}
	}
	return false
}

func removeChild(path string, isSuffixEmpty bool, suffixList []string) error {
	if path == "" {
		// fail silently to retain compatibility with previous behavior
		// of RemoveAll. See issue 28830.
		return nil
	}

	// The rmdir system call does not permit removing ".",
	// so we don't permit it either.
	if endsWithDot(path) {
		return unix.EINVAL
	}

	pathLen := len(path)
	if path[pathLen-1] != slash {
		path += slashStr
	}

	var (
		respSize  int
		dirF      *os.File
		err       error
		nameList  []string
		childname string
		childPath string
		removeErr error
		numErr    int
	)

	for {
		dirF, err = os.Open(path)
		if errors.Is(err, fs.ErrNotExist) {
			// If path does not exist, Fail silently
			return nil
		}
		if err != nil {
			return err
		}

		for {
			numErr = 0
			nameList, err = dirF.Readdirnames(reqSize)

			if err != nil {
				_ = dirF.Close()

				if errors.Is(err, io.EOF) {
					return nil
				}

				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}

				return err
			}

			respSize = len(nameList)

			for _, childname = range nameList {
				// only specify suffix and not match file skip current entry
				if !isSuffixEmpty && !isNeedRemove(childname, suffixList) {
					continue
				}

				childPath = path + childname
				err = os.RemoveAll(childPath)
				if err != nil {
					if removeErr == nil {
						removeErr = err
					}
					numErr += 1
					log.Println(
						"[rmWrapper-Warning]Failed to remove path:", childPath,
						"and err:", err.Error())
				}
			}

			// If we can delete any entry, break to start new iteration.
			// Otherwise, we discard current names, continue get next entries and try deleting them.
			if numErr != reqSize {
				break
			}
		}

		// Removing files from the directory may have caused
		// the OS to reshuffle it. Simply calling Readdirnames
		// again may skip some entries. The only reliable way
		// to avoid this is to close and re-open the
		// directory. See issue 20841.
		_ = dirF.Close()

		// Finish when the end of the directory is reached
		if respSize < reqSize {
			break
		}
	}

	if removeErr != nil {
		return removeErr
	}

	return nil
}

// endsWithDot reports whether the final component of path is ".".
func endsWithDot(path string) bool {
	if path == "." {
		return true
	}
	if len(path) >= 2 && path[len(path)-1] == '.' && IsPathSeparator(path[len(path)-2]) {
		return true
	}
	return false
}

// IsPathSeparator reports whether c is a directory separator character.
func IsPathSeparator(c uint8) bool {
	return PathSeparator == c
}
