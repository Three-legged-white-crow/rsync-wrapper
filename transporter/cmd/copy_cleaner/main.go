package main

import (
	"errors"
	"flag"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"time"

	copy2 "transporter/internal/copy"
	"transporter/pkg/checksum"
	"transporter/pkg/exit_code"
	"transporter/pkg/filesystem"
)

const (
	emptyValue         = "empty"
	slash              = '/'
	slashStr           = "/"
	waitNFSCliUpdate   = 5
	waitNFSCcliLimit   = 5
	defaultLimtReadDir = 100
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
		"src",
		emptyValue,
		"src path relative to the src mount point")

	destTempDirRelativePath := flag.String(
		"dest-temp-dir",
		emptyValue,
		"dest temp dir path relative to the dest mount point")

	destFinalDirRelativePath := flag.String(
		"dest-final-dir",
		emptyValue,
		"dest final dir path relative to the dest mount point")

	isExcludeSrcDir := flag.Bool(
		"exclude-src",
		false,
		"exclude src dir(src must is dir)")

	trackFileRelativePath := flag.String(
		"track-file",
		emptyValue,
		"track file relative to the dest mount point")

	isDebug := flag.Bool(
		"debug",
		false,
		"enable debug mode")

	flag.Parse()

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)

	log.Println("[copyClean-Info]New copy-clean request: srcRelativePath:", *srcRelativePath,
		"destTempDirRelativePath:", *destTempDirRelativePath,
		"destFinalDirRelativePath:", *destFinalDirRelativePath,
		"srcMountPath:", *srcMountPath,
		"destMountPath:", *destMountPath,
		"isExcludeSrcDir:", *isExcludeSrcDir,
		"trackFileRelativePath:", *trackFileRelativePath,
		"isDebug:", *isDebug,
	)

	log.Println("[copyClean-Info]Start basic check")
	log.Println("[copyClean-Info]Start basic check format")
	var (
		isPathAvailable  bool
		srcPath          string
		destTempDirPath  string
		destFinalDirPath string
		err              error
		isCleanTrackFile bool
		trackFilePath    string
		exitCode         int
	)

	isPathAvailable = filesystem.CheckDirPathFormat(*srcMountPath)
	if !isPathAvailable {
		log.Println("[copyClean-Error]Unavailable format of src mount point:", *srcMountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckDirPathFormat(*destMountPath)
	if !isPathAvailable {
		log.Println("[copyClean-Error]Unavailable format of dest mount point:", *destMountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *srcRelativePath == emptyValue {
		log.Println("[copyClean-Error]Unavailable format of src relative path:", srcRelativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *destTempDirRelativePath == emptyValue {
		log.Println("[copyClean-Error]Unavailable format of dest temp dir relative path:", *destTempDirRelativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *destFinalDirRelativePath == emptyValue {
		log.Println("[copyClean-Error]Unavailable format of dest final dir relative path:", *destFinalDirRelativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *trackFileRelativePath != emptyValue {
		isCleanTrackFile = true
	}

	// not need check err, because format of mount point has already been checked above
	srcPath, _ = filesystem.AbsolutePath(*srcMountPath, *srcRelativePath)
	destTempDirPath, _ = filesystem.AbsolutePath(*destMountPath, *destTempDirRelativePath)
	destFinalDirPath, _ = filesystem.AbsolutePath(*destMountPath, *destFinalDirRelativePath)
	if destFinalDirPath[len(destFinalDirPath)-1] != slash {
		destFinalDirPath += slashStr
	}

	if destTempDirPath[len(destTempDirPath)-1] != slash {
		destTempDirPath += slashStr
	}

	var srcPath1 string = srcPath
	srcLen := len(srcPath1)
	if srcLen > 1 {
		if srcPath1[srcLen-1] == slash {
			srcPath1 = srcPath1[:srcLen-1]
		}
	}

	isPathAvailable = filesystem.CheckFilePathFormat(srcPath1)
	if !isPathAvailable {
		log.Println("[copyClean-Error]Unavailable src path:", srcPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[copyClean-Info]Check basic format...OK")

	if isCleanTrackFile {
		trackFilePath, _ = filesystem.AbsolutePath(*destMountPath, *trackFileRelativePath)
		log.Println("[copyClean-Info]Need clean track file:", trackFilePath)
		isPathAvailable = filesystem.CheckFilePathFormat(trackFilePath)
		if !isPathAvailable {
			log.Println("[copyClean-Error]Unavailable track file path:", trackFilePath)
			os.Exit(exit_code.ErrInvalidArgument)
		}
		log.Println("[copyClean-Info]Check track file format...OK")
	}

	if !(*isDebug) {
		log.Println("[copyClean-Info]Start check mount filesystem")
		err = filesystem.IsMountPath(*srcMountPath)
		if err != nil {
			log.Println("[copyClean-Info]Failed to check src mount filesystem:", *srcMountPath,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		err = filesystem.IsMountPath(*destMountPath)
		if err != nil {
			log.Println("[copyClean-Info]Failed to check dest mount filesystem:", *destMountPath,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
		log.Println("[copyClean-Info]Check mount filesystem...OK")
	}

	log.Println("[copyClean-Info]End basic check")
	/*
		clean stage:
			- clean final dest;
			- clean temp dest;
			- clean track file (if exist);
			- clean checksum file (if exist);
			- clean flag file;

		if src is file:
			- if final dest is exist
				- try remove file;

		if src is dir:
			- if final dest is exist
				- if exclude src dir
					- range src dir and search child dir is exist in final dest dir
						- try remove child dir from final dest dir;
				- if not exclude src dir
					- remove src dir from final dest dir;

		- remove temp dest dir;
		- remove complate flag file when copy file;
		- remove track file (if exist);
		- remove checksum file (if exist) when copy file;
		- complate clean
	*/

	log.Println("[copyClean-Info]Start check src is exist")
	var (
		srcInfo      os.FileInfo
		retryStatNum int
	)
	for {
		if retryStatNum >= waitNFSCcliLimit {
			log.Println("[copyClean-Error]Src path:", srcPath1,
				"is not exist, retry stat num:", retryStatNum)
			os.Exit(exit_code.ErrNoSuchFileOrDir)
		}

		srcInfo, err = os.Stat(srcPath1)
		if err == nil {
			break
		}

		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[copyClean-Error]Failed to stat src path:", srcPath1,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		time.Sleep(waitNFSCliUpdate * time.Second)
		retryStatNum += 1
	}
	log.Println("[copyClean-Info]Check src path is exist...Exist")

	log.Println("[copyClean-Info]Start check final dest dir is exist")
	var (
		destFinalDirInfo    os.FileInfo
		isDestFinalDirExist bool
	)
	destFinalDirInfo, err = os.Stat(destFinalDirPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[copyClean-Error]Failed to stat final dest dir:", destFinalDirPath, "and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		isDestFinalDirExist = false
		log.Println("[copyClean-Info]Check final dest dir...NotExist")
	} else {
		if !destFinalDirInfo.IsDir() {
			log.Println("[copyClean-Info]Check final dest dir...Exist, but is file")
			log.Println("[copyClean-Error]Final dest dir is a exist file")
			os.Exit(exit_code.ErrNotDirectory)
		}

		isDestFinalDirExist = true
	}

	if isDestFinalDirExist {
		if srcInfo.IsDir() {
			// case: cp -rf /home/dir1/* /home/dir2/ ==> /home/dir2/*
			if *isExcludeSrcDir {
				err = removeChild(srcPath, destFinalDirPath)
				if err != nil {
					log.Println("[copyClean-Warning]Get err when clean dest final dir:", destFinalDirPath,
						"reference src dir", srcPath,
						"and err:", err.Error())
				} else {
					log.Println("[copyClean-Info]Succeed to clean dest final:", destFinalDirPath, "with exclude src dir")
				}
			} else {
				// case: cp -rf /home/dir1 /home/dir2/ ==> /home/dir2/dir1
				srcDirName := srcInfo.Name()
				destFinalSubDirPath := destFinalDirPath + srcDirName
				err = os.RemoveAll(destFinalSubDirPath)
				if err != nil {
					log.Println("[copyClean-Warning]Failed to remove dest final sub dir:", destFinalSubDirPath, "and err:", err.Error())
				} else {
					log.Println("[copyClean-Info]Succeed to remove dest final sub dir:", destFinalSubDirPath)
				}
			}

		} else {
			srcFileName := srcInfo.Name()
			destFinalFileName := destFinalDirPath + srcFileName
			err = os.Remove(destFinalFileName)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					log.Println("[copyClean-Info]Dest final file:", destFinalFileName, "is not exist")
				} else {
					log.Println("[copyClean-Warning]Failed to remove dest final file:", destFinalFileName, "and err:", err.Error())
				}
			} else {
				log.Println("[copyClean-Info]Succeed to remove dest final file:", destFinalFileName)
			}

			// try remove checksum file
			destFinalChecksumFileName := destFinalFileName + checksum.MD5Suffix
			err = os.Remove(destFinalChecksumFileName)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					log.Println("[copyClean-Info]Not exist checksum file:", destFinalChecksumFileName)
				} else {
					log.Println("[copyClean-Warning]Failed to remove checksum file:", destFinalChecksumFileName, "and err:", err.Error())
				}
			} else {
				log.Println("[copyClean-Info]Succeed to remove checksum file:", destFinalChecksumFileName)
			}

			// try clean complete flag file
			completeFlagFilePath := path.Clean(destTempDirPath) + copy2.CompleteFlagFileName
			err = os.Remove(completeFlagFilePath)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					log.Println("[copyClean-Info]Not exist complete flag file:", completeFlagFilePath)
				} else {
					log.Println("[copyClean-Warning]Get err:", err.Error(), "when clean complete flag file")
				}
			} else {
				log.Println("[copyClean-Info]Succeed to clean complete flag file:", completeFlagFilePath)
			}
		}

	}

	// clean temp dir
	err = os.RemoveAll(destTempDirPath)
	if err != nil {
		log.Println("[copyClean-Warning]Get err:", err.Error(), "when clean temp dir:", destTempDirPath)
	} else {
		log.Println("[copyClean-Info]Succeed to clean temp dir:", destTempDirPath)
	}

	// clean track file
	if isCleanTrackFile {
		err = os.Remove(trackFilePath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				log.Println("[copyClean-Info]Not exist track file:", trackFilePath)
			} else {
				log.Println("[copyClean-Warning]Get err:", err.Error(), "when remove track file:", trackFilePath)
			}
		} else {
			log.Println("[copyClean-Info]Succeed to remove track file:", trackFilePath)
		}
	} else {
		log.Println("[copyClean-Info]Not need clean track file")
	}

	log.Println("[copyClean-Info]Complete all clean, exit")
	os.Exit(exit_code.Succeed)
}

func removeChild(rfcDir, targetDir string) error {
	rfcDirLen := len(rfcDir)
	targetDirLen := len(targetDir)

	if rfcDirLen == 0 || targetDirLen == 0 {
		return fs.ErrNotExist
	}

	if rfcDir[rfcDirLen-1] != slash {
		rfcDir += slashStr
	}

	if targetDir[targetDirLen-1] != slash {
		targetDir += slashStr
	}

	var (
		respSize        int
		rfcDirF         *os.File
		err             error
		nameList        []string
		childname       string
		targetChildPath string
	)

	for {
		rfcDirF, err = os.Open(rfcDir)
		if err != nil {
			log.Println("[copyClean-Error]Failed to open reference dir:", rfcDir, "and err:", err.Error())
			return err
		}

		nameList, err = rfcDirF.Readdirnames(defaultLimtReadDir)
		if err != nil {
			_ = rfcDirF.Close()

			if errors.Is(err, io.EOF) {
				return nil
			}

			log.Println("[copyClean-Error]Failed to readdirnames of reference dir:", rfcDir, "and err:", err.Error())
			return err
		}

		respSize = len(nameList)
		for _, childname = range nameList {
			targetChildPath = targetDir + childname
			err = os.RemoveAll(targetChildPath)
			if err != nil {
				log.Println("[copyClean-Warning]Failed to remove target child:", targetChildPath, "and err:", err.Error())
			}
		}

		_ = rfcDirF.Close()

		// Finish when the end of the directory is reached
		if respSize < defaultLimtReadDir {
			break
		}
	}

	return nil
}
