package main

import (
	"errors"
	"flag"
	"io"
	"io/fs"
	"log"
	"os"

	"transporter/pkg/exit_code"
	"transporter/pkg/filesystem"
)

const (
	emptyValue         = "empty"
	slash              = '/'
	defaultLimtReadDir = 100
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

	srcInfo, err := os.Stat(srcPath1)
	if err != nil {
		log.Println(
			"[mvWrapper-Error]Failed to get src info with stat:", srcPath1,
			"and err:", err.Error())

		filesystem.Exit(err)
	}

	// src is dir
	if srcInfo.IsDir() {
		// case: mv src/* dest
		if *isExcludeSrcDir {
			var (
				srcDirPath string = srcPath1 + "/"
				destDirPath string = *destPath
			)
			if destDirPath[len(destDirPath)-1] != slash {
				destDirPath += "/"
			}

			var destInfo os.FileInfo
			destInfo, err = os.Stat(*destPath)
			if err != nil {
				log.Println(
					"[mvWrapper-Error]Failed to get dest info with stat:", *destPath,
					", isExcludeSrcDir:", *isExcludeSrcDir,
					"and err:", err.Error())

				filesystem.Exit(err)
			}

			if !destInfo.IsDir() {
				log.Println("[mvWrapper-Error]Dest is not dir:", *destPath)
				os.Exit(exit_code.ErrNotDirectory)
			}

			log.Println("[mvWrapper-Info]Src is dir and exclude src dir, rename start")
			log.Println("[mvWrapper-Info]Src is dir and exclude src dir, read dir name list is start")

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
					if err != nil {
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
						continue
					}
					// new file is exist
					log.Println("[mvWrapper-Error]New file:", newFilePath, "is already exist at dest dir:", destDirPath)
					os.Exit(exit_code.ErrFileIsExists)
				}
			}
			log.Println("[mvWrapper-Info]Src is dir and exclude src dir, renmae end")

		}else {
			log.Println("[mvWrapper-Info]Src is dir and include src dir, rename start")

			err = os.Rename(srcPath1, *destPath)
			if err != nil {
				log.Println(
					"[mvWrapper-Error]Failed to rename src dir:", srcPath1,
					"to dest:", *destPath,
					"isExcludeSrcDir:", *isExcludeSrcDir,
					"and err:", err.Error())

				filesystem.Exit(err)
			}

			log.Println("[mvWrapper-Info]Src is dir and include src dir, rename end")
		}

	}else {
		// src is file
		if (*srcPath)[srcLen-1] == slash {
			log.Println("[mvWrapper-Error]Src is a file, but path is dir:", *srcPath)
			os.Exit(exit_code.ErrInvalidArgument)
		}

		log.Println("[mvWrapper-Info]Src is file, rename start")
		err = os.Rename(srcPath1, *destPath)
		if err != nil {
			log.Println(
				"[mvWrapper-Error]Failed to rename src file:", srcPath1,
				"to dest:", *destPath,
				"and err:", err.Error())

			filesystem.Exit(err)
		}

		log.Println("[mvWrapper-Info]Src is file, rename end")
	}

	log.Println(
		"[mvWrapper-Info]Succeed to move src:", srcPath1,
		"to dest:", destPath,
		"isExcludeSrcDir", *isExcludeSrcDir)
	os.Exit(exit_code.Succeed)
}
