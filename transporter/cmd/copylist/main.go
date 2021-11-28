package main

import (
	"flag"
	"log"
	"os"

	"transporter/pkg/exit_code"
	"transporter/pkg/filesystem"
)

const (
	emptyValue       = "empty"
	slash            = '/'
	slashStr         = "/"
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

	inputRecordFile := flag.String(
		"record-file-in",
		emptyValue,
		"input record file path relative dest mount point, use for copy file list")

	outputRecordFile := flag.String(
		"record-file-out",
		emptyValue,
		"output record file path relative dest mount point, use for copy file list")

	isIgnoreSrcNotExist := flag.Bool(
		"ignore-src-not-exist",
		false,
		"if src file is not exist will skip error and record to file that 'record-file-out' specified, use for copy file list")

	isIgnoreSrcIsDir := flag.Bool(
		"ignore-src-dir",
		false,
		"if src file is dir will skip error and record to file that 'record-file-out' specified")

	isIgnoreDestIsExistDir := flag.Bool(
		"ignore-dest-dir",
		false,
		"if dest is exist dir will skip error and record to file that 'record-file-out' specified")

	isOverwriteDestFile := flag.Bool(
		"overwrite-dest-file",
		false,
		"overwirte dest exist file, effective for file or file list, if dest is dir, do nothing")

	isGenerateChecksumFile := flag.Bool(
		"generate-checksum-file",
		false,
		"generate checksum file to same dir as dest, effective for file or file list, if dest is dir, do nothing")
	flag.Parse()

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)

	log.Println("[copylist-Info]New transporter request, srcMountPath:", *srcMountPath,
		"destMountPath:", *destMountPath,
		"inputRecordFile:", *inputRecordFile,
		"outputRecordFile:", *outputRecordFile,
		"isIgnoreSrcNotExist:", *isIgnoreSrcNotExist,
		"isIgnoreSrcIsDir:", *isIgnoreSrcIsDir,
		"isIgnoreDestIsExistDir:", *isIgnoreDestIsExistDir,
		"isOverwriteDestFile:", *isOverwriteDestFile,
		"isGenerateChecksumFile:", *isGenerateChecksumFile,
	)

	log.Println("[copylist-Info]Start check")
	log.Println("[copylist-Info]Start check format")

	var (
		isPathAvailable   bool
		inRecordFilePath  string
		outRecordFilePath string
		err               error
	)
	isPathAvailable = filesystem.CheckDirPathFormat(*srcMountPath)
	if !isPathAvailable {
		log.Println("[copylist-Error]Unavailable format of src mount point:", *srcMountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckDirPathFormat(*destMountPath)
	if !isPathAvailable {
		log.Println("[copylist-Error]Unavailable format of dest mount point:", *destMountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	// not need check err, because format of mount point has already been checked above
	inRecordFilePath, _ = filesystem.AbsolutePath(*destMountPath, *inputRecordFile)
	outRecordFilePath, _ = filesystem.AbsolutePath(*destMountPath, *outputRecordFile)

	isPathAvailable = filesystem.CheckFilePathFormat(inRecordFilePath)
	if !isPathAvailable {
		log.Println("[copylist-Error]Unavailable format of input record file:", inRecordFilePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckFilePathFormat(outRecordFilePath)
	if !isPathAvailable {
		log.Println("[copylist-Error]Unavailable format of output record file:", outRecordFilePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[copylist-Info]Check format...OK")

	log.Println("[copylist-Info]Start check mount filesystem")
	err = filesystem.IsMountPath(*srcMountPath)
	if err != nil {
		log.Println(
			"[copylist-Error]Failed to check src mount filesystem:", *srcMountPath,
			"and err:", err.Error())
		filesystem.Exit(err)
	}

	err = filesystem.IsMountPath(*destMountPath)
	if err != nil {
		log.Println(
			"[copylist-Error]Failed to check dest mount filesystem:", *destMountPath,
			"and err:", err.Error())
		filesystem.Exit(err)
	}
	log.Println("[copylist-Info]Check mount filesystem...OK")

	// todo: check input file exist -> ENOEXIST

	// todo: check output file exist -> trancunt

	// todo: parse input file

	// todo: build src file and dest file

	// todo: check src /dest file exist -> record to output file

	// todo: check src/ dest is dir -> record to output file

	// todo: rsync src to dest -> get stderr of rsync and convert exit code

	// todo: checksum -> if not equal -> rm dest -> retry rsync -> checksum -> if not equal -> record

	// todo: is record something to output file? -> yes -> 252

	// todo: exit


	os.Exit(exit_code.Succeed)

}
