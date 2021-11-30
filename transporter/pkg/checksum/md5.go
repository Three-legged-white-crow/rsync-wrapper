package checksum

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
)

const (
	readBuf      = 1024
	MD5Algorithm = "md5"
	MD5Suffix    = ".md5"
)

var ErrNotEqual = errors.New("checksum result of src and dest is not equal")

func MD5(filePath string) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := md5.New()
	buf := make([]byte, readBuf)

	var n int
	for {
		n, err = f.Read(buf)
		if err == io.EOF {
			break
		}

		h.Write(buf[:n])
	}

	return h.Sum(nil), nil

}

func Compare(src, dest []byte) bool {
	if len(src) != len(dest) {
		return false
	}

	for i, v := range src {
		if v != dest[i] {
			return false
		}
	}

	return true
}

func MD5File(path string, res []byte) error {

	dst := make([]byte, hex.EncodedLen(len(res)))
	hex.Encode(dst, res)
	reshex := string(dst)

	log.Println("[Checksum-Info]Hexadecimal encoding checksum string:", reshex)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(dst)
	if err != nil {
		return err
	}

	return nil
}

func MD5Checksum(srcFilePath, destFilePath string, isGenerateChecksumFile bool) error {
	var srcChecksum []byte
	var err error
	srcChecksum, err = MD5(srcFilePath)
	if err != nil {
		return err
	}

	var destChecksum []byte
	destChecksum, err = MD5(destFilePath)
	if err != nil {
		return err
	}

	isEqual := Compare(srcChecksum, destChecksum)
	if !isEqual {
		return ErrNotEqual
	}

	if isGenerateChecksumFile {
		md5DestFilePath := destFilePath + MD5Suffix
		err = MD5File(md5DestFilePath, destChecksum)
		if err != nil {
			return err
		}
	}
	return nil
}