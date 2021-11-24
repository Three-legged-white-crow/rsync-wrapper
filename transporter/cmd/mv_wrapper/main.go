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
	defaultLimtReadDir = 100
	waitNFSCliUpdate   = 5
	waitNFSCcliLimit   = 3
)

func main() {

	srcPath := flag.String("src", emptyValue, "src abs path")
	destPath := flag.String("dest", emptyValue, "dest abs path")
	isExcludeSrcDir := flag.Bool("exclude-src", false, "exclude src dir(src must is dir), default is false")
	flag.Parse()

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)
	log.Println(
		"[mvWrapper-Info]New mv request, src:", *srcPath,
		"dest:", *destPath,
		"isExcludeSrcDir", *isExcludeSrcDir)
	log.Println("[mvWrapper-Info]Start check")

	var srcPath1 string = *srcPath
	srcLen := len(srcPath1)
	if srcLen > 1 {
		if srcPath1[srcLen-1] == slash {
			srcPath1 = srcPath1[:srcLen-1]
		}
	}

	log.Println("[mvWrapper-Info]Start check path format")
	isPathAvailable := filesystem.CheckFilePathFormat(srcPath1)
	if !isPathAvailable {
		log.Println("[mvWrapper-Error]Unavailable src path:", *srcPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckDirPathFormat(*destPath)
	if !isPathAvailable {
		log.Println("[mvWrapper-Error]Unavailable dest path:", *destPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[mvWrapper-Info]Check path format...OK")

	// sleep and retry avoid NFS client cache not update
	var (
		err         error
		srcInfo     os.FileInfo
		srcRetryNum int
	)
	log.Println("[mvWrapper-Info]Start check src path is exist")
	for {
		if srcRetryNum >= waitNFSCcliLimit {
			log.Println("[mvWrapper-Error]src path:", srcPath1,"is not exist, retry stat num:", srcRetryNum)
			os.Exit(exit_code.ErrNoSuchFileOrDir)
		}

		time.Sleep(waitNFSCliUpdate * time.Second)
		srcInfo, err = os.Stat(srcPath1)
		if err == nil {
			break
		}

		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[mvWrapper-Error]Failed to stat src path:", srcPath1, "and err:", err.Error())
			filesystem.Exit(err)
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
		destInfo, err = os.Stat(*destPath)
		if err == nil {
			log.Println("[mvWrapper-Info]Check dest path is exist...Exist")
			isDestExist = true
			break
		}

		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[mvWrapper]Failed to stat dest path:", *destPath, "and err:", err.Error())
			filesystem.Exit(err)
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
				srcDirPath string = srcPath1 + "/"
				destDirPath string = *destPath
			)
			if destDirPath[len(destDirPath)-1] != slash {
				destDirPath += "/"
			}

			log.Println("[mvWrapper-Info]Rename start, src is dir and exclude src dir:", srcPath1)
			log.Println("[mvWrapper-Info]Start read names of src dir")
			var sf *os.File
			sf, err = os.Open(srcPath1)
			if err != nil {
				log.Println(
					"[mvWrapper-Error]Failed to open src dir:", srcPath1,
					", isExcludeSrcDir:", *isExcludeSrcDir,
					"and err:", err.Error())
				filesystem.Exit(err)
			}
			defer sf.Close()

			var (
				nameList []string
				oldFilePath  string
				newFilePath string
			)
			for {
				nameList, err = sf.Readdirnames(defaultLimtReadDir)
				if err != nil {
					if errors.Is(err, io.EOF) {
						log.Println(
							"[mvWrapper-Info]Get EOF when read dir name list of src dir:", srcPath1,
							", isExcludeSrcDir:", *isExcludeSrcDir)
						break
					}
					log.Println(
						"[mvWrapper-Error]Failed to read name list of src dir:", srcPath1,
						", isExcludeSrcDir:", *isExcludeSrcDir,
						"and err:", err.Error())

					filesystem.Exit(err)
				}

				for _, name := range nameList {
					newFilePath = destDirPath + name
					_, err = os.Stat(newFilePath)
					if err == nil {
						// new file is exist
						log.Println("[mvWrapper-Error]New file:", newFilePath, "is already exist at dest dir:", destDirPath)
						os.Exit(exit_code.ErrFileIsExists)
					}

					// get err when stat new file
					if !errors.Is(err, fs.ErrNotExist) {
						log.Println(
							"[mvWrapper-Error]Failed to stat new file:", newFilePath,
							", isExcludeSrcDir:", *isExcludeSrcDir,
							"and err:", err.Error())
						filesystem.Exit(err)
					}
					// new file is not exit, allow rename
					oldFilePath = srcDirPath + name
					err = os.Rename(oldFilePath, newFilePath)
					if err != nil {
						log.Println("[mvWrapper-Error]Failed to rename old file:", oldFilePath,
							"to new file:", newFilePath,
							", isExcludeSrcDir:", *isExcludeSrcDir,
							"and err:", err.Error())
						filesystem.Exit(err)
					}
				}
			}
			log.Println("[mvWrapper-Info]Rename end, src is dir and exclude src dir:", srcPath1)

		}else {
			if !isDestExist || destInfo == nil {
				log.Println("[mvWrapper-Info]Rename start, src is dir:", srcPath1,
					"and include src dir, dest is not exist:", *destPath)
				err = os.Rename(srcPath1, *destPath)
				if err != nil {
					log.Println(
						"[mvWrapper-Error]Failed to rename src dir:", srcPath1,
						"to dest:", *destPath,
						"isExcludeSrcDir:", *isExcludeSrcDir,
						"and err:", err.Error())

					filesystem.Exit(err)
				}
				log.Println("[mvWrapper-Info]Rename end, src is dir:",srcPath1,
					"and include src dir, dest is not exist:", *destPath)
			}else {
				if !destInfo.IsDir() {
					log.Println("[mvWrapper-Error]Src is dir and include src dir, but dest is a exist file")
					os.Exit(exit_code.ErrNotDirectory)
				}

				srcFileName := filepath.Base(srcPath1)
				destDirPath := *destPath
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

					filesystem.Exit(err)
				}
				log.Println("[mvWrapper-Info]Rename end, src is dir:", srcPath1,
					"and include src dir, dest already exist, new path:", newFilePath)
			}
		}

	}else {
		// src is file and dest is not exist
		if !isDestExist || destInfo == nil {
			log.Println("[mvWrapper-Info]Rename start, src is file:", srcPath1, "dest is not exist:", *destPath)
			err = os.Rename(srcPath1, *destPath)
			if err != nil {
				log.Println(
					"[mvWrapper-Error]Failed to rename src file:", srcPath1,
					"to not exist dest file:", *destPath,
					"and err:", err.Error())
				filesystem.Exit(err)
			}
			log.Println("[mvWrapper-Info]Rename end, src is file:", srcPath1, "dest is not exist:", *destPath)

		}else {
			// src is file but dest is a exist file -> EEXIST
			if !destInfo.IsDir() {
				log.Println("[mvWrapper-Error]Src is file:", srcPath1, "but dest is a exist file:", *destPath)
				os.Exit(exit_code.ErrFileIsExists)
			}

			// src is file and dest is exist dir
			destDirPath := *destPath
			if destDirPath[len(destDirPath)-1] != slash {
				destDirPath += "/"
			}
			srcFileName := filepath.Base(srcPath1)
			newFilePath := destDirPath + srcFileName

			_, err = os.Stat(newFilePath)
			if err == nil {
				// at dest already exist file that same name to src file
				log.Println("[mvWrapper-Error]New file:", newFilePath, "is already exist in dest dir:", destDirPath)
				os.Exit(exit_code.ErrFileIsExists)
			}

			log.Println("[mvWrapper-Info]Rename start, src is file:", srcPath1, "dest is exist dir, new file:", newFilePath)
			err = os.Rename(srcPath1, newFilePath)
			if err != nil {
				log.Println("[mvWrapper-Error]Failed to rename src file:", srcPath1,
					"to new file:", newFilePath,
					"and err:", err.Error())
				filesystem.Exit(err)
			}
			log.Println("[mvWrapper-Info]Rename end, src is file:", srcPath1, "dest is exist dir, new file:", newFilePath)
		}
	}

	log.Println(
		"[mvWrapper-Info]Succeed to move src:", srcPath1,
		"to dest:", destPath,
		"isExcludeSrcDir", *isExcludeSrcDir)
	os.Exit(exit_code.Succeed)
}
