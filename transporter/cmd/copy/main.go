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

	"transporter/pkg/checksum"
	"transporter/pkg/client"
	"transporter/pkg/exit_code"
	"transporter/pkg/filesystem"
	"transporter/pkg/rsync_wrapper/dir"
	"transporter/pkg/rsync_wrapper/file"
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

	isReportProgress := flag.Bool(
		"progress",
		false,
		"report progress of the transmission, must used with 'report-addr' flag")

	isReportStderr := flag.Bool(
		"stderr",
		false,
		"report std error content, must used with 'report-addr' flag")

	addrReport := flag.String(
		"report-addr",
		emptyValue,
		"addr for report progress info or error message")

	intervalReport := flag.Int(
		"report-interval",
		0,
		"interval for report progress info, time unit is second, must positive integer")

	isExcludeSrcDir := flag.Bool(
		"exclude-src",
		false,
		"exclude src dir(src must is dir)")

	isOverwriteDestFile := flag.Bool(
		"overwrite-dest-file",
		false,
		"overwirte dest exist file, effective for file or file list, if dest is dir, do nothing")

	isGenerateChecksumFile := flag.Bool(
		"generate-checksum-file",
		false,
		"generate checksum file to same dir as dest, effective for file or file list, if dest is dir, do nothing")

	fileSuffixForChecksum := flag.String(
		"checksum-suffix",
		emptyValue,
		"suffix of file that need checksum")

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

	log.Println("[copy-Info]New transporter request: srcRelativePath:", *srcRelativePath,
		"destTempDirRelativePath:", *destTempDirRelativePath,
		"destFinalDirRelativePath:", *destFinalDirRelativePath,
		"srcMountPath:", *srcMountPath,
		"destMountPath:", *destMountPath,
		"isReportProgress:", *isReportProgress,
		"isReportStderr:", *isReportStderr,
		"reportAddress:", *addrReport,
		"reportInterval(second):", *intervalReport,
		"isExcludeSrcDir:", *isExcludeSrcDir,
		"isOverwriteDestFile:", *isOverwriteDestFile,
		"isGenerateChecksumFile:", *isGenerateChecksumFile,
		"fileSuffixForChecksum:", *fileSuffixForChecksum,
		"trackFileRelativePath:", *trackFileRelativePath,
		"isDebug:", *isDebug,
	)

	log.Println("[copy-Info]Start basic check")
	log.Println("[copy-Info]Start basic check format")
	var (
		isPathAvailable        bool
		srcPath                string
		destTempDirPath        string
		err                    error
		isCreateTrackFile      bool
		exitCode               int
		isChecksumSuffixEmpty  bool
		isFileNeedChecksum     bool
		checksumFileSuffixList []string
	)

	isPathAvailable = filesystem.CheckDirPathFormat(*srcMountPath)
	if !isPathAvailable {
		log.Println("[copy-Error]Unavailable format of src mount point:", *srcMountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckDirPathFormat(*destMountPath)
	if !isPathAvailable {
		log.Println("[copy-Error]Unavailable format of dest mount point:", *destMountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *srcRelativePath == emptyValue {
		log.Println("[copy-Error]Unavailable format of src relative path:", srcRelativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *destTempDirRelativePath == emptyValue {
		log.Println("[copy-Error]Unavailable format of dest temp dir relative path:", *destTempDirRelativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *destFinalDirRelativePath == emptyValue {
		log.Println("[copy-Error]Unavailable format of dest final dir relative path:", *destFinalDirRelativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *fileSuffixForChecksum == emptyValue {
		isChecksumSuffixEmpty = true
		log.Println("[copy-Info]Not specify checksum suffix, not checksum")
	}else {
		checksumFileSuffixList = strings.Split(*fileSuffixForChecksum, slashStr)
	}

	if *trackFileRelativePath != emptyValue {
		isCreateTrackFile = true
	}

	// not need check err, because format of mount point has already been checked above
	srcPath, _ = filesystem.AbsolutePath(*srcMountPath, *srcRelativePath)
	destTempDirPath, _ = filesystem.AbsolutePath(*destMountPath, *destTempDirRelativePath)
	destFinalDirPath, _ := filesystem.AbsolutePath(*destMountPath, *destFinalDirRelativePath)
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
		log.Println("[copy-Error]Unavailable src path:", srcPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[copy-Info]Check basic format...OK")

	if !(*isDebug) {
		log.Println("[copy-Info]Start check mount filesystem")
		err = filesystem.IsMountPath(*srcMountPath)
		if err != nil {
			log.Println("[copy-Info]Failed to check src mount filesystem:", *srcMountPath,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		err = filesystem.IsMountPath(*destMountPath)
		if err != nil {
			log.Println("[copy-Info]Failed to check dest mount filesystem:", *destMountPath,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
		log.Println("[copy-Info]Check mount filesystem...OK")
	}

	log.Println("[copy-Info]End basic check")

	/*
		if src is not exist -> ENOENT

		src is file:
			- check track file format
			- create temp dest dir
			- rsync src file to temp dir:
				- if failed to rsync -> get exit code from stderr of rsync -> exit with code
				- if succeed to rsync -> filter file with suffix
					- if match -> checksum
						- if equal -> -> generate result file
						- if not equal -> rm file from temp dir -> retry rsync
					- create final dest dir
					- rename file from temp dir to final dir
						- if file is exist in final dir
							- if overwrite -> rename directly
							- if not overwrite -> EEXIST
						- if file not exist -> rename directly
					- create track file
					- rm temp dir

		src is dir:
			- ExcludeSrcDir -> src + "/"
			- if not ExcludeSrcDir -> use src directly
			- rsync src to temp dest dir
				- if failed to rsync -> get exit code from stderr of rsync -> accord exit code retry or not
				- if succeed to rsync -> succeed
			- if need report progress and stderr -> start goroutine to report
	 */

	log.Println("[copy-Info]Start check src is exist")
	var (
		srcInfo  os.FileInfo
		retryNum int
	)
	for {
		if retryNum >= waitNFSCcliLimit {
			log.Println("[copy-Error]Src path:", srcPath1,
				"is not exist, retry stat num:", retryNum)
			os.Exit(exit_code.ErrNoSuchFileOrDir)
		}

		time.Sleep(waitNFSCliUpdate * time.Second)
		srcInfo, err = os.Stat(srcPath1)
		if err == nil {
			break
		}

		if errors.Is(err, fs.ErrNotExist) {
			log.Println("[copy-Error]Failed to stat src path:", srcPath1,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		retryNum += 1
	}
	log.Println("[copy-Info]Check src path is exist...Exist")

	log.Println("[copy-Info]Start check temp dest dir is exist")
	var destTempDirInfo os.FileInfo
	destTempDirInfo, err = os.Stat(destTempDirPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[copy-Error]Failed to stat temp dest dir:", destTempDirPath, "and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		log.Println("[copy-Info]Check temp dest dir...NotExist")
		err = filesystem.CheckOrCreateDir(destTempDirPath)
		if err != nil {
			log.Println("[copy-Error]Failed to create temp dest dir:", destTempDirPath,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
		log.Println("[copy-Info]Succeed to create temp dest dir:", destTempDirPath)
	}else {
		if !destTempDirInfo.IsDir() {
			log.Println("[copy-Info]Check temp dest dir...Exist, but is file")
			log.Println("[copy-Error]Temp dest dir is a exist file")
			os.Exit(exit_code.ErrNotDirectory)
		}
	}

	// src is dir
	if srcInfo.IsDir() {

		// case: cp -rf /home/dir/* /home/dir/
		if *isExcludeSrcDir && ((srcPath1 + slashStr) == destFinalDirPath) {
			log.Println(
				"[copy-Error]The source and destination are the same file, parent dir:",
				destFinalDirPath)
			os.Exit(exit_code.ErrSrcAndDstAreSameFile)
		}

		// case: cp -rf /home/dir /home/
		if !(*isExcludeSrcDir) && (srcPath1 == (destFinalDirPath + srcInfo.Name())) {
			log.Println(
				"[copy-Error]The source and destination are the same file, parent dir:",
				destFinalDirPath)
			os.Exit(exit_code.ErrSrcAndDstAreSameFile)
		}

		if !(*isExcludeSrcDir) && ((srcPath1 + slashStr) == destFinalDirPath) {
			log.Println("[copy-Error]Cannot copy a directory into itself, dir:",
				srcPath1)
			os.Exit(exit_code.ErrDirectoryNestedItself)
		}

			log.Println("[copy-Info]Src is dir, ready copy")
		if *isExcludeSrcDir {
			log.Println("[copy-Info]Start check final dest dir has same name file or dir that wait copy")
			var isDestFinalDirAvailable bool
			isDestFinalDirAvailable, err = checkDestFinalDir(srcPath1, destFinalDirPath)
			if err != nil {
				log.Println("[copy-Error]Faild to check final dest dir is available:", destFinalDirPath,
					"and err:", err.Error())
				exitCode = exit_code.ExitCodeConvertWithErr(err)
				os.Exit(exitCode)
			}

			if !isDestFinalDirAvailable {
				log.Println("[copy-Error]Unavailable final dest dir:", destFinalDirPath,
					", there is same name file at src dir:", srcPath1)
				os.Exit(exit_code.ErrFileIsExists)
			}
			log.Println("[copy-Info]Check final dest dir...Available")

			srcPath1 += slashStr
		}

		rc := client.NewReportClient()

		reqCopyDir := dir.ReqRun{
			SrcPath:          srcPath1,
			DestPath:         destTempDirPath,
			IsReportProgress: *isReportProgress,
			IsReportStderr:   *isReportStderr,
			ReportClient:     rc,
			ReportInterval:   *intervalReport,
			ReportAddr:       *addrReport,
		}

		startTime := time.Now().String()
		log.Println("[copy-Info]Dir copy, start at:", startTime)
		exitCode = dir.Run(reqCopyDir)
		endTime := time.Now().String()
		log.Println("[copy-Info]Dir copy, end at:", endTime)

		log.Println("[copy-Warning]Src is dir, copy end with exit code:", exitCode)

		// sleep a moment for wait all goroutine exit
		time.Sleep(5 * time.Second)
		os.Exit(exitCode)
	}

	// src is file
	log.Println("[copy-Info]Src is file, start format check")
	isPathAvailable = filesystem.CheckFilePathFormat(srcPath1)
	if !isPathAvailable {
		log.Println("[copy-Error]Unavailable src file path:", srcPath1)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[copy-Info]Check src path format...OK")


	var trackFilePath string
	if isCreateTrackFile {
		trackFilePath, _ = filesystem.AbsolutePath(*destMountPath, *trackFileRelativePath)
		log.Println("[copy-Info]Need create track file:", trackFilePath)
		isPathAvailable = filesystem.CheckFilePathFormat(trackFilePath)
		if !isPathAvailable {
			log.Println("[copy-Error]Unavailable track file path:", trackFilePath)
			os.Exit(exit_code.ErrInvalidArgument)
		}
		log.Println("[copy-Info]Check track file format...OK")
	}

	log.Println("[copy-Info]Start check final dest dir is exist")
	var destFinalDirInfo os.FileInfo
	destFinalDirInfo, err = os.Stat(destFinalDirPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[copy-Error]Failed to stat final dest dir:", destFinalDirPath, "and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		log.Println("[copy-Info]Check final dest dir...NotExist")
		err = filesystem.CheckOrCreateDir(destFinalDirPath)
		if err != nil {
			log.Println("[copy-Error]Failed to create final dest dir:", destFinalDirPath,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
		log.Println("[copy-Info]Succeed to create final dest dir:", destFinalDirPath)
	}else {
		if !destFinalDirInfo.IsDir() {
			log.Println("[copy-Info]Check final dest dir...Exist, but is file")
			log.Println("[copy-Error]Final dest dir is a exist file")
			os.Exit(exit_code.ErrNotDirectory)
		}
	}

	// case: cp /home/dir/file /home/dir/ or cp /home/dir/file /home/dir/file
	if (srcPath1 == (destFinalDirPath + srcInfo.Name())) || (srcPath1 + slashStr == destFinalDirPath) {
		log.Println("[copy-Error]The source and destination are the same file, file:", srcPath1)
		os.Exit(exit_code.ErrSrcAndDstAreSameFile)
	}

	log.Println("[copy-Info]Start copy file, Step 1 -> copy file from src:", srcPath1,
		"to temp dest dir:", destTempDirPath)
	fileName := srcInfo.Name()
	destTempFileName := destTempDirPath + fileName
	destTempCheckFileName := destTempFileName + checksum.MD5Suffix
	destFinalFileName := destFinalDirPath + fileName
	destFinalCheckFileName := destFinalFileName + checksum.MD5Suffix
	exitCode = file.CopyFile(srcPath1, destTempFileName)
	if exitCode != exit_code.Succeed {
		log.Println("[copy-Error]Failed to copy(1) file from src:", srcPath1,
			"to dest:", destTempFileName,
			"and exit code:", exitCode)
		os.Exit(exitCode)
	}

	log.Println("[copy-Info]Succeed to copy(1) file from src:", srcPath1,
		"to dest:", destTempFileName)

	if !isChecksumSuffixEmpty && isNeedChecksum(fileName, checksumFileSuffixList) {
		isFileNeedChecksum = true

		log.Println("[copy-Info]Start checksum(1), src:", srcPath1, "dir:", destTempFileName)
		err = checksum.MD5Checksum(srcPath1, destTempFileName, *isGenerateChecksumFile)
		if err != nil {
			// internal retry again
			err = os.Remove(destTempFileName)
			if err != nil {
				if !errors.Is(err, fs.ErrNotExist) {
					log.Println(
						"[copy-Error]Internal retry at copy file, failed to remove temp dest file:", destTempFileName,
						"and err:", err.Error())
					exitCode = exit_code.ExitCodeConvertWithErr(err)
					os.Exit(exitCode)
				}
			}
			log.Println("[copy-Info]Internal retry at copy file, succeed to remove temp dest file")

			log.Println("[copy-Info]Internal retry at copy file, start copy(2) file from src:", srcPath1,
				"to dest:", destTempFileName)
			exitCode = file.CopyFile(srcPath1, destTempFileName)
			if exitCode != exit_code.Succeed {
				log.Println(
					"[copy-Error]Internal retry at copy file, failed to copy(2) file from src:", srcPath1,
					"to dest:", destTempFileName,
					"and exit code:", exitCode)
				os.Exit(exitCode)
			}
			log.Println("[copy-Info]Internal retry at copy file, succeed to copy(2) file from src:", srcPath1,
				"to dest:", destTempFileName)

			log.Println("[copy-Info]Internal retry at copy file, start to checksum(2) file src:", srcPath1,
				"with dest:", destTempFileName)
			err = checksum.MD5Checksum(srcPath1, destTempFileName, *isGenerateChecksumFile)
			if err != nil {
				log.Println(
					"[copy-Error]Internal retry at copy file, failed to checksum(2) again, and err:",
					err.Error())
				os.Exit(exit_code.ErrChecksumRefuse)
			}
			log.Println("[copy-Info]Internal retry at copy file, succeed to checksum(2) file src:", srcPath1,
				"with dest:", destTempFileName)
		}
		log.Println("[copy-Info]Succeed to checksum src:", srcPath1, "dest:", destTempFileName)

	}


	log.Println("[copy-Info]Start copy file, ->Step 2-> copy file from temp dest:", destTempFileName,
		"to final dest:", destFinalFileName)
	var destFinalFileInfo os.FileInfo
	destFinalFileInfo, err = os.Stat(destFinalFileName)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[copy-Error]Failed to stat final dest file:", destFinalFileName,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
	}

	if destFinalFileInfo != nil {
		if destFinalFileInfo.IsDir() {
			log.Println("[copy-Error]Final dest file exist but is dir:", destFinalFileName)
			os.Exit(exit_code.ErrIsDirectory)
		}

		if !(*isOverwriteDestFile) {
			log.Println("[copy-Error]Final dest file is exist file, but not overwrite:", destFinalFileName)
			os.Exit(exit_code.ErrFileIsExists)
		}
	}

	// rename file from temp dir to final dir
	err = os.Rename(destTempFileName, destFinalFileName)
	if err != nil {
		log.Println("[copy-Error]Failed to rename dest file from temp:", destTempFileName,
			"to final:", destFinalFileName, "and err:", err.Error())
		exitCode = exit_code.ExitCodeConvertWithErr(err)
		os.Exit(exitCode)
	}
	log.Println(
		"[copy-Info]Succeed to rename file from temp dest:", destTempFileName,
		"to final dest:", destFinalFileName)

	if isFileNeedChecksum && *isGenerateChecksumFile {
		err = os.Rename(destTempCheckFileName, destFinalCheckFileName)
		if err != nil {
			log.Println("[copy-Error]Failed to rename dest checksum file from temp:", destTempCheckFileName,
				"to final:", destFinalCheckFileName, "and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
		log.Println(
			"[copy-Info]Succeed to rename checksum file from temp dest:", destTempCheckFileName,
			"to final dest:", destFinalCheckFileName)
	}

	if isCreateTrackFile {
		log.Println("[copy-Info]Start create track file:", trackFilePath)
		err = filesystem.CheckOrCreateFile(trackFilePath, false)
		if err != nil {
			log.Println("[copy-Error]Failed to check or create track file:", trackFilePath,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
		log.Println("[copy-Info]Succeed to create track file:", trackFilePath)
	}

	log.Println("[copy-Info]Remove temp dest dir:", destTempDirPath)
	err = os.RemoveAll(destTempDirPath)
	if err != nil {
		log.Println("[copy-Waring]Failed to remove temp dest dir:", destTempDirPath,
			"and err:", err.Error())
	}else {
		log.Println("[copy-Info]Succeed to remove temp dest dir:", destTempDirPath)
	}

	log.Println("[copy-Info]Copy file is end with exit code:", exit_code.ErrCopyFileSucceed)
	os.Exit(exit_code.ErrCopyFileSucceed)

}

func isNeedChecksum(fileName string, fileSuffixList []string) bool {
	if fileSuffixList == nil {
		return false
	}

	var	isMatch bool
	for _, suffix := range fileSuffixList {
		isMatch, _ = filepath.Match(suffix, fileName)
		if isMatch {
			return true
		}
	}
	return false
}

func checkDestFinalDir(srcDirPath, destDirPath string) (bool, error) {
	var (
		sf  *os.File
		err error
	)
	sf, err = os.Open(srcDirPath)
	if err != nil {
		log.Println(
			"[copy-Error]Failed to open src dir:", srcDirPath,
			"and err:", err.Error())
		return false, err
	}
	defer sf.Close()

	var (
		nameList []string
		newFilePath string
	)
	for {
		nameList, err = sf.Readdirnames(defaultLimtReadDir)
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Println(
					"[copy-Info]Get EOF when read dir name list of src dir:", srcDirPath)
				break
			}
			log.Println(
				"[copy-Error]Failed to read name list of src dir:", srcDirPath,
				"and err:", err.Error())

			return false, err
		}

		for _, name := range nameList {
			newFilePath = destDirPath + name
			_, err = os.Stat(newFilePath)
			if err == nil {
				// new file is exist
				log.Println("[copy-Error]New file:", newFilePath,
					"is already exist at dest dir:", destDirPath)
				return false, nil
			}

			// get err when stat new file
			if !errors.Is(err, fs.ErrNotExist) {
				log.Println(
					"[copy-Error]Failed to stat new file:", newFilePath,
					"and err:", err.Error())
				return false, err
			}

		}
	}

	return true, nil
}