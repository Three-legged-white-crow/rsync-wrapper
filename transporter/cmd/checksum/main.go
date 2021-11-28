package main

import (
	"flag"
	"log"
	"os"

	"transporter/pkg/checksum"
	"transporter/pkg/exit_code"
	"transporter/pkg/filesystem"
)

const (
	emptyValue   = "empty"
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

	checksumAlgorithm := flag.String(
		"checksum",
		checksum.MD5Algorithm,
		"checksum algorithm")

	isGenerateChecksumFile := flag.Bool(
		"dest-generate",
		false,
		"generate checksum file to dest path")

	flag.Parse()

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)
	log.Println("[checksum-Info]New checksum request, srcRelativePath:", *srcRelativePath,
		"destRelativePath:", *destRelativePath,
		"srcMountPath:", *srcMountPath,
		"destMountPath:", *destMountPath,
		"algorithm:", *checksumAlgorithm,
		"isGenerateChecksumFile", *isGenerateChecksumFile)
	log.Println("[checksum-Info]Start check")
	log.Println("[checksum-Info]Start check format")

	var (
		isPathAvailable bool
		srcFilePath     string
		destFilePath    string
		err             error
	)

	isPathAvailable = filesystem.CheckDirPathFormat(*srcMountPath)
	if !isPathAvailable {
		log.Println("[checksum-Error]Unavailable format of src mount point:", *srcMountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckDirPathFormat(*destMountPath)
	if !isPathAvailable {
		log.Println("[checksum-Error]Unavailable format of dest mount point:", *destMountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *srcRelativePath == emptyValue {
		log.Println("[checksum-Error]Unavailable format of src file relative path:", srcRelativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *destRelativePath == emptyValue {
		log.Println("[checksum-Error]Unavailable format of dest file relative path:", *destRelativePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	srcFilePath, err = filesystem.AbsolutePath(*srcMountPath, *srcRelativePath)
	if err != nil {
		log.Println("[checksum-Error]Unavailable format of src mount point:", *srcMountPath,
			"or src file relative path:", *srcRelativePath, err.Error())
		os.Exit(exit_code.ErrInvalidArgument)
	}

	destFilePath, err = filesystem.AbsolutePath(*destMountPath, *destRelativePath)
	if err != nil {
		log.Println("[checksum-Error]Unavailable format of dest mount point:", *destMountPath,
			"or dest file relative path:", *destRelativePath, err.Error())
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckFilePathFormat(srcFilePath)
	if !isPathAvailable {
		log.Println("[checksum-Error]Unavailable src:", srcFilePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckFilePathFormat(destFilePath)
	if isPathAvailable {
		log.Println("[checksum-Error]Unavailable dest:", destFilePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[checksum-Info]Check path format...OK")

	if *checksumAlgorithm != checksum.MD5Algorithm {
		log.Println("[checksum-Error]Only support md5 at now")
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[checksum-Info]Check algorithm...OK")

	log.Println("[checksum-Info]Start check mount filesystem")
	err = filesystem.IsMountPath(*srcMountPath)
	if err != nil {
		log.Println("[checksum-Error]Failed to check mount of src mount point:", *srcMountPath,
			"and err:", err.Error())
		filesystem.Exit(err)
	}

	err = filesystem.IsMountPath(*destMountPath)
	if err != nil {
		log.Println("[checksum-Error]Failed to check mount of dest mount point:", *destMountPath,
			"and err:", err.Error())
		filesystem.Exit(err)
	}
	log.Println("[checksum-Info]Check mount filesystem...OK")
	log.Println("[checksum-Info]End check")

	log.Println("[checksum-Info]Start checksum")
	var srcChecksum []byte
	srcChecksum, err = checksum.MD5Checksum(srcFilePath)
	if err != nil {
		log.Println("[checksum-Error]Failed to generate checksum of src file, err:", err.Error())
		filesystem.Exit(err)
	}

	var destChecksum []byte
	destChecksum, err = checksum.MD5Checksum(destFilePath)
	if err != nil {
		log.Println("[checksum-Error]Failed to generate checksum of dest file, err:", err.Error())
		filesystem.Exit(err)
	}

	isEqual := checksum.Compare(srcChecksum, destChecksum)
	if !isEqual {
		log.Println("[checksum-Error]Result of checksum with src file and dest file is not equal!!!")
		os.Exit(exit_code.ErrChecksumRefuse)
	}

	log.Println("[checksum-Info]Result of checksum with src file and dest file is equal...")

	if *isGenerateChecksumFile {
		md5DestFilePath := destFilePath + checksum.MD5Suffix
		err = checksum.MD5File(md5DestFilePath, destChecksum)
		if err != nil {
			log.Println("[checksum-Error]Failed to write checksum file, err:", err.Error())
			os.Exit(exit_code.ErrIOError)
		}
		log.Println("[checksum-Info]Succeed to write checksum file:", md5DestFilePath)
	}

	log.Println("[checksum-Info]End checksum")
	os.Exit(exit_code.Succeed)
}
