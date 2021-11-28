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

var ErrWriteChecksumContent = errors.New("failed to write full content")

func MD5Checksum(filePath string) ([]byte, error) {
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
	l := hex.Encode(dst, res)
	reshex := string(dst)

	log.Println("[Checksum-Info]Hexadecimal encoding checksum string:", reshex)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := f.Write(dst)
	if err != nil {
		return err
	}

	if n != l {
		return ErrWriteChecksumContent
	}

	return nil
}
