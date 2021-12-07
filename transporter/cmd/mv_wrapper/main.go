package main

import (
	"errors"
	"flag"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"transporter/pkg/exit_code"
	"transporter/pkg/filesystem"
)

const (
	emptyValue         = "empty"
	slash              = '/'
	slashStr           = "/"
	defaultLimtReadDir = 1024
	waitNFSCliUpdate   = 5
	waitNFSCcliLimit   = 5
)

func main() {

	srcMountPath := flag.String(
		"src-mount",
		emptyValue,
		"mount point path of src")

	destMountPath := flag.String(
		"dest-mount",
		emptyValue,
		"mount point path of dest")

	srcRelativePath := flag.String(
		"src-relative",
		emptyValue,
		"src file path relative to the src mount point")

	destRelativePath := flag.String(
		"dest-relative",
		emptyValue,
		"dest file path relative to the dest mount point")

	isExcludeSrcDir := flag.Bool(
		"exclude-src",
		false,
		"exclude src dir(src must is dir)")

	isDebug := flag.Bool(
		"debug",
		false,
		"enable debug mode")

	flag.Parse()

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)
	log.Println(
		"[mvWrapper-Info]New mv request, srcRelativePath:", *srcRelativePath,
		"destRelativePath:", *destRelativePath,
		"srcMountPath:", *srcMountPath,
		"destMountPath:", *destMountPath,
		"isExcludeSrcDir", *isExcludeSrcDir,
		"isDebug:", *isDebug,
	)
	log.Println("[mvWrapper-Info]Start check")

	log.Println("[mvWrapper-Info]Start check format")

	var (
		isPathAvailable bool
		srcPath         string
		destPath        string
		err             error
	)

	isPathAvailable = filesystem.CheckDirPathFormat(*srcMountPath)
	if !isPathAvailable {
		log.Println("[mvWrapper-Error]Unavailable format of src mount point:", *srcMountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckDirPathFormat(*destMountPath)
	if !isPathAvailable {
		log.Println("[mvWrapper-Error]Unavailable format of dest mount point:", *destMountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *srcRelativePath == emptyValue {
		log.Println("[mvWrapper-Error]Unavailable format of src relative path:", srcRelativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *destRelativePath == emptyValue {
		log.Println("[mvWrapper-Error]Unavailable format of dest relative path:", *destRelativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	// not need check err, because format of mount point has already been checked above
	srcPath, _ = filesystem.AbsolutePath(*srcMountPath, *srcRelativePath)
	destPath, _ = filesystem.AbsolutePath(*destMountPath, *destRelativePath)

	var srcPath1 string = srcPath
	srcLen := len(srcPath1)
	if srcLen > 1 {
		if srcPath1[srcLen-1] == slash {
			srcPath1 = srcPath1[:srcLen-1]
		}
	}

	isPathAvailable = filesystem.CheckFilePathFormat(srcPath1)
	if !isPathAvailable {
		log.Println("[mvWrapper-Error]Unavailable src path:", srcPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckDirPathFormat(destPath)
	if !isPathAvailable {
		log.Println("[mvWrapper-Error]Unavailable dest path:", destPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[mvWrapper-Info]Check path format...OK")

	var exitCode int
	if !(*isDebug) {
		log.Println("[mvWrapper-Info]Start check src mount filesystem")
		err = filesystem.IsMountPath(*srcMountPath)
		if err != nil {
			log.Println("[mvWrapper-Error]Failed to check src mount filesystem, err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
		log.Println("[mvWrapper-Info]Check src mount filesystem...OK")

		log.Println("[mvWrapper-Info]Start check dest mount filesystem")
		err = filesystem.IsMountPath(*destMountPath)
		if err != nil {
			log.Println("[mvWrapper-Error]Failed to check dest mount filesystem, err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
		log.Println("[mvWrapper-Info]Check dest mount filesystem...OK")
	}

	// sleep and retry avoid NFS client cache not update
	var (
		srcInfo     os.FileInfo
		srcRetryNum int
	)
	log.Println("[mvWrapper-Info]Start check src path is exist")
	for {
		if srcRetryNum >= waitNFSCcliLimit {
			log.Println("[mvWrapper-Error]Src path:", srcPath1,
				"is not exist, retry stat num:", srcRetryNum)
			os.Exit(exit_code.ErrNoSuchFileOrDir)
		}

		time.Sleep(waitNFSCliUpdate * time.Second)
		srcInfo, err = os.Stat(srcPath1)
		if err == nil {
			break
		}

		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[mvWrapper-Error]Failed to stat src path:", srcPath1,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		srcRetryNum += 1
	}
	log.Println("[mvWrapper-Info]Check src path is exist...Exist")

	var (
		isDestExist  bool = false
		destInfo     os.FileInfo
		destRetryNum int
	)
	log.Println("[mvWrapper-Info]Start check dest path is exist")
	for {
		// dest path allow not exist
		if destRetryNum >= waitNFSCcliLimit {
			log.Println("[mvWrapper-Info]Check dest path is exist...NotExist")
			break
		}

		time.Sleep(waitNFSCliUpdate * time.Second)
		destInfo, err = os.Stat(destPath)
		if err == nil {
			log.Println("[mvWrapper-Info]Check dest path is exist...Exist")
			isDestExist = true
			break
		}

		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[mvWrapper]Failed to stat dest path:", destPath, "and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		destRetryNum += 1
	}

	log.Println("[mvWrapper-Info]End check")
	log.Println("[mvWrapper-Info]Start move")
	/*
		src is dir:
			- if exclude src dir(like cmd:'mv src/* dest'):
				- if dest not exist -> ENOENT;
				- if dest is exit:
					- if dest is not dir -> ENOTDIR;
					- if dest is dir -> read dir names of src -> build new file name:
						- check new file name is exist in dest dir:
							- if new file exist in dest dir -> EEXIST;
							- if get err when stat new file -> EACCES/EPERM/ErrSystem(custom);
							- if new file not exist -> rename(src/file, dest/file) ;
				 - remove src dir(empty dir)

			- if include src dir:
				- if dest not exist -> rename(src, dest);
				- if dest is exist:
					- if dest is file -> ENOTDIR;
					- if dest is dir -> rename(src, dest/src);

		src is file:
			- if dest not exist -> rename(src, dest);
			- if dest exist:
				- if dest is file -> EEXIST;
				- if dest is dir -> build new file name:
					- check new file name is exist in dest dir:
						- if new file exist in dest dir -> EEXIST;
						- if new file not exist -> rename(src, dest/src);

	*/

	// src is dir
	if srcInfo.IsDir() {
		// case: mv /home/dir /home/dir
		var destPath1 string = destPath
		destLen := len(destPath1)
		if destLen > 1 {
			if destPath1[destLen-1] == slash {
				destPath1 = destPath1[:destLen-1]
			}
		}

		if !(*isExcludeSrcDir) && (srcPath1 == destPath1) {
			log.Println("[mvWrapper-Error]Cannot copy a directory into itself, dir:",
				srcPath1)
			os.Exit(exit_code.ErrDirectoryNestedItself)
		}

		// case: mv /home/dir/* /home/dir/
		if *isExcludeSrcDir && (srcPath1 == destPath1) {
			log.Println(
				"[mvWrapper-Error]The source and destination are the same file, parent dir:",
				srcPath1)
			os.Exit(exit_code.ErrSrcAndDstAreSameFile)
		}

		// case: mv /home/dir /home
		var destPath2 string = destPath
		destLen2 := len(destPath2)
		if destPath2[destLen2-1] != slash {
			destPath2 += slashStr
		}
		if !(*isExcludeSrcDir) && (srcPath1 == destPath2+srcInfo.Name()) {
			log.Println(
				"[mvWrapper-Error]The source and destination are the same file:", srcPath1)
			os.Exit(exit_code.ErrSrcAndDstAreSameFile)
		}

		// case: mv src/* dest
		if *isExcludeSrcDir {

			if !isDestExist {
				log.Println("[mvWrapper-Error]Src is dir and exclude src dir, but dest is not exist")
				os.Exit(exit_code.ErrNoSuchFileOrDir)
			}

			if destInfo == nil {
				log.Println("[mvWrapper-Error]Src is dir and exclude src dir, but dest is not exist")
				os.Exit(exit_code.ErrNoSuchFileOrDir)
			}

			if !destInfo.IsDir() {
				log.Println("[mvWrapper-Error]Src is dir and exclude src dir, but dest is not dir")
				os.Exit(exit_code.ErrNotDirectory)
			}

			var (
				srcDirPath  string = srcPath1 + "/"
				destDirPath string = destPath
			)
			if destDirPath[len(destDirPath)-1] != slash {
				destDirPath += "/"
			}

			log.Println("[mvWrapper-Info]Rename start, src is dir and exclude src dir:", srcPath1)
			log.Println("[mvWrapper-Info]Start read names of src dir")
			err = renameChild(srcDirPath, destDirPath)
			if err != nil {
				log.Println("[mvWrapper-Error]Failed to rename file from src dir:", srcDirPath,
					"to dest dir:", destDirPath, "and err:", err.Error())
				exitCode = exit_code.ExitCodeConvertWithErr(err)
				os.Exit(exitCode)
			}
			log.Println("[mvWrapper-Info]Rename end, src is dir and exclude src dir:", srcPath1)

		} else {
			if !isDestExist || destInfo == nil {
				log.Println("[mvWrapper-Info]Rename start, src is dir:", srcPath1,
					"and include src dir, dest is not exist:", destPath)
				err = os.Rename(srcPath1, destPath)
				if err != nil {
					log.Println(
						"[mvWrapper-Error]Failed to rename src dir:", srcPath1,
						"to dest:", destPath,
						"isExcludeSrcDir:", *isExcludeSrcDir,
						"and err:", err.Error())

					exitCode = exit_code.ExitCodeConvertWithErr(err)
					os.Exit(exitCode)
				}
				log.Println("[mvWrapper-Info]Rename end, src is dir:", srcPath1,
					"and include src dir, dest is not exist:", destPath)
			} else {
				if !destInfo.IsDir() {
					log.Println("[mvWrapper-Error]Src is dir and include src dir, but dest is a exist file")
					os.Exit(exit_code.ErrNotDirectory)
				}

				srcFileName := filepath.Base(srcPath1)
				destDirPath := destPath
				if destDirPath[len(destDirPath)-1] != slash {
					destDirPath += "/"
				}
				newFilePath := destDirPath + srcFileName
				log.Println("[mvWrapper-Info]Rename start, src is dir:", srcPath1,
					"and include src dir, dest already exist, new path:", newFilePath)
				err = os.Rename(srcPath1, newFilePath)
				if err != nil {
					log.Println(
						"[mvWrapper-Error]Failed to rename src dir:", srcPath1,
						"to dest dir:", newFilePath,
						"isExcludeSrcDir:", *isExcludeSrcDir,
						"and err:", err.Error())

					exitCode = exit_code.ExitCodeConvertWithErr(err)
					os.Exit(exitCode)
				}
				log.Println("[mvWrapper-Info]Rename end, src is dir:", srcPath1,
					"and include src dir, dest already exist, new path:", newFilePath)
			}
		}

	} else {
		// src is file and dest is not exist
		if !isDestExist || destInfo == nil {
			log.Println("[mvWrapper-Info]Rename start, src is file:", srcPath1,
				"dest is not exist:", destPath)
			err = os.Rename(srcPath1, destPath)
			if err != nil {
				log.Println(
					"[mvWrapper-Error]Failed to rename src file:", srcPath1,
					"to not exist dest file:", destPath,
					"and err:", err.Error())
				exitCode = exit_code.ExitCodeConvertWithErr(err)
				os.Exit(exitCode)
			}
			log.Println("[mvWrapper-Info]Rename end, src is file:", srcPath1,
				"dest is not exist:", destPath)

		} else {
			// case: mv /home/dir/file /home/dir/file
			if srcPath1 == destPath {
				log.Println("[mvWrapper-Error]The source and destination are the same file, file:", srcPath1)
				os.Exit(exit_code.ErrSrcAndDstAreSameFile)
			}

			// src is file but dest is a exist file -> EEXIST
			if !destInfo.IsDir() {
				log.Println("[mvWrapper-Error]Src is file:", srcPath1,
					"but dest is a exist file:", destPath)
				os.Exit(exit_code.ErrFileIsExists)
			}

			// src is file and dest is exist dir
			destDirPath := destPath
			if destDirPath[len(destDirPath)-1] != slash {
				destDirPath += "/"
			}
			srcFileName := filepath.Base(srcPath1)
			newFilePath := destDirPath + srcFileName

			_, err = os.Stat(newFilePath)
			if err == nil {
				// at dest already exist file that same name to src file
				log.Println("[mvWrapper-Error]New file:", newFilePath,
					"is already exist in dest dir:", destDirPath)
				os.Exit(exit_code.ErrFileIsExists)
			}

			log.Println("[mvWrapper-Info]Rename start, src is file:", srcPath1,
				"dest is exist dir, new file:", newFilePath)
			err = os.Rename(srcPath1, newFilePath)
			if err != nil {
				log.Println("[mvWrapper-Error]Failed to rename src file:", srcPath1,
					"to new file:", newFilePath,
					"and err:", err.Error())
				exitCode = exit_code.ExitCodeConvertWithErr(err)
				os.Exit(exitCode)
			}
			log.Println("[mvWrapper-Info]Rename end, src is file:", srcPath1,
				"dest is exist dir, new file:", newFilePath)
		}
	}

	log.Println(
		"[mvWrapper-Info]Succeed to move src:", srcPath1,
		"to dest:", destPath,
		"isExcludeSrcDir", *isExcludeSrcDir)
	os.Exit(exit_code.Succeed)
}

func renameChild(srcDir, destDir string) error {
	if len(srcDir) == 0 || len(destDir) == 0 {
		return fs.ErrNotExist
	}

	srcDirLen := len(srcDir)
	if srcDir[srcDirLen-1] != slash {
		srcDir += slashStr
	}

	destDirLen := len(destDir)
	if destDir[destDirLen-1] != slash {
		destDir += slashStr
	}

	var (
		respSize     int
		srcDirF      *os.File
		err          error
		nameList     []string
		childname    string
		childOldPath string
		childNewPath string
	)

	for {
		srcDirF, err = os.Open(srcDir)
		if err != nil {
			log.Println("[mvWrapper-Error]Failed to open src dir:", srcDir, "and err:", err.Error())
			return err
		}

		nameList, err = srcDirF.Readdirnames(defaultLimtReadDir)
		if err != nil {
			_ = srcDirF.Close()

			if errors.Is(err, io.EOF) {
				return nil
			}

			log.Println("[mvWrapper-Error]Failed to readdirnames of path:", srcDir, "and err:", err.Error())
			return err
		}

		respSize = len(nameList)
		for _, childname = range nameList {
			childNewPath = destDir + childname
			_, err = os.Stat(childNewPath)
			if err == nil {
				log.Println("[mvWrapper-Error]Path is exist:", childNewPath)
				return fs.ErrExist
			}

			if !errors.Is(err, fs.ErrNotExist) {
				log.Println("[mvWrapper-Error]Failed to stat path:", childNewPath, "and err:", err.Error())
				return err
			}

			childOldPath = srcDir + childname
			err = os.Rename(childOldPath, childNewPath)
			if err != nil {
				log.Println("[mvWrapper-Error]Failed to rename from old:", childOldPath,
					"to new:", childNewPath, "and err:", err.Error())
				return err
			}
		}

		// Removing files from the directory may have caused
		// the OS to reshuffle it. Simply calling Readdirnames
		// again may skip some entries. The only reliable way
		// to avoid this is to close and re-open the
		// directory. See issue 20841.
		_ = srcDirF.Close()

		// Finish when the end of the directory is reached
		if respSize < defaultLimtReadDir {
			break
		}
	}

	return nil
}
