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
	md5Algorithm = "md5"
	md5Suffix    = ".md5"
)

func main() {

	srcFilePath := flag.String("src", "", "src file abs path")
	destFilePath := flag.String("dest", "", "dest file abs path")
	checksumAlgorithm := flag.String("checksum", "md5", "checksum algorithm")
	isGenerateChecksumFile := flag.Bool("dest-generate", false, "generate checksum file to dest path")
	flag.Parse()

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)
	log.Println("[checksum-Info]New checksum request, src:", *srcFilePath,
		"dest:", *destFilePath,
		"algorithm:", *checksumAlgorithm,
		"isGenerateChecksumFile", *isGenerateChecksumFile)
	log.Println("[checksum-Info]Start check")

	var isPathAvailable bool
	isPathAvailable = filesystem.CheckFilePathFormat(*srcFilePath)
	if !isPathAvailable {
		log.Println("[checksum-Error]Unavailable src:", *srcFilePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckFilePathFormat(*destFilePath)
	if isPathAvailable {
		log.Println("[checksum-Error]Unavailable dest:", *destFilePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[checksum-Info]Check path format...OK")

	if *checksumAlgorithm != md5Algorithm {
		log.Println("[checksum-Error]Only support md5 at now")
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[checksum-Info]Check algorithm...OK")

	var err error
	err = filesystem.IsMountPath(*srcFilePath)
	if err != nil {
		log.Println("[checksum-Error]Failed to check mount of src, err:", err.Error())
		filesystem.Exit(err)
	}

	err = filesystem.IsMountPath(*destFilePath)
	if err != nil {
		log.Println("[checksum-Error]Failed to check mount of dest, err:", err.Error())
		filesystem.Exit(err)
	}
	log.Println("[checksum-Info]Check mount filesystem...OK")
	log.Println("[checksum-Info]End check")

	log.Println("[checksum-Info]Start checksum")
	var srcChecksum []byte
	srcChecksum, err = checksum.MD5Checksum(*srcFilePath)
	if err != nil {
		log.Println("[checksum-Error]Failed to generate checksum of src file, err:", err.Error())
		filesystem.Exit(err)
	}

	var destChecksum []byte
	destChecksum, err = checksum.MD5Checksum(*destFilePath)
	if err != nil {
		log.Println("[checksum-Error]Failed to generate checksum of dest file, err:", err.Error())
		filesystem.Exit(err)
	}

	isEqual := checksum.Compare(srcChecksum, destChecksum)
	if !isEqual {
		log.Println("[checksum-Error]Result of checksum with src file and dest file is not equal!!!")
		os.Exit(exit_code.ErrSystem)
	}

	log.Println("[checksum-Info]Result of checksum with src file and dest file is equal...")

	if *isGenerateChecksumFile {
		md5DestFilePath := *destFilePath + md5Suffix
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
