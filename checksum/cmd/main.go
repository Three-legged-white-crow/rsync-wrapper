package main

import (
	"flag"
	"log"
	"os"

	"checksum"
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

	if len(*srcFilePath) == 0 {
		log.Println("[Checksum]Not specify src file")
		os.Exit(checksum.ErrInvalidArgument)
	}

	if len(*destFilePath) == 0 {
		log.Println("[Checksum]Not specify dest file")
		os.Exit(checksum.ErrInvalidArgument)
	}

	if *checksumAlgorithm != md5Algorithm {
		log.Println("[Checksum]Only support md5 at now")
		os.Exit(checksum.ErrInvalidArgument)
	}

	srcChecksum, err := checksum.MD5Checksum(*srcFilePath)
	if err != nil {
		log.Println("[Checksum]Failed to generate checksum of src file, err:", err.Error())
		os.Exit(checksum.ErrSystem)
	}

	destChecksum, err := checksum.MD5Checksum(*destFilePath)
	if err != nil {
		log.Println("[Checksum]Failed to generate checksum of dest file, err:", err.Error())
		os.Exit(checksum.ErrSystem)
	}

	isEqual := checksum.Compare(srcChecksum, destChecksum)
	if !isEqual {
		log.Println("[Checksum]Result of checksum with src file and dest file is not equal!!!")
		os.Exit(checksum.ErrSystem)
	}

	log.Println("[Checksum]Result of checksum with src file and dest file is equal...")

	if *isGenerateChecksumFile {
		md5DestFilePath := *destFilePath + md5Suffix
		err = checksum.MD5File(md5DestFilePath, destChecksum)
		if err != nil {
			log.Println("[Checksum]Failed to write checksum file, err:", err.Error())
			os.Exit(checksum.ErrIOError)
		}
		log.Println("[Checksum]Succeed to write checksum file:", md5DestFilePath)
	}

	os.Exit(checksum.Succeed)
}
