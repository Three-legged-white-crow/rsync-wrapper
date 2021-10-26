package rsync_wrapper

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync/atomic"
	"time"

	"transporter/pkg/client"
)

const (
	errNonNumeric            = "ascii char is non-numeric"
	progressBufSize          = 32

	// progressLine format: #xfr99
	progressLineFirstChar    = '#'
	progressNumStartIndex    = 4
	progressReporterInterval = 5
)

// readStdout read content of stdout with pipe, parsed progress.
func readStdout(ctx context.Context, reader io.Reader, progressNum *uint32) {
	var (
		num      uint32
		l        []byte
		isPrefix bool
		err      error
	)

	r := bufio.NewReaderSize(reader, progressBufSize)

	for {
		select {
		case <-ctx.Done():
			return

		default:

		}

		// use read method of reader, in this case, use read method of *File.
		l, isPrefix, err = r.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}

			continue
		}

		// means content's length of line at output is large than progress buffer size
		if isPrefix {
			continue
		}

		// read empty line
		if len(l) == 0 {
			continue
		}

		// not progress line
		if l[0] != progressLineFirstChar {
			continue
		}

		num, err = atoi(l[progressNumStartIndex:])
		if err != nil {
			continue
		}

		atomic.StoreUint32(progressNum, num)
	}
}

// atoi convert bytes of number string format to type uint32.
func atoi(strb []byte) (uint32, error) {
	var n uint32
	if len(strb) == 1 {

		if strb[0] > '9' || strb[0] < '0' {
			return 0, errors.New(errNonNumeric)
		}

		nb := strb[0] - '0'
		n = uint32(nb)
		return n, nil
	}

	for _, c := range strb {
		c -= '0'
		if c > 9 {
			return 0, errors.New(errNonNumeric)
		}
		n = n*10 + uint32(c)
	}

	return n, nil
}

func reportProgress(ctx context.Context, progressNum *uint32, addr string, rc *client.ReportClient) {
	t := time.NewTicker(progressReporterInterval * time.Second)

	var (
		currentProgressNum uint32
		reqContent         reqResult
	)

	for {
		select {
		case <-ctx.Done():
			t.Stop()
			return

		case <-t.C:
			currentProgressNum = atomic.LoadUint32(progressNum)

			reqContent.Count = int64(currentProgressNum)
			reqContentB, err := json.Marshal(&reqContent)
			if err != nil {
				continue
			}

			err = rc.Report(addr, client.ContentType, reqContentB)
			if err != nil {
				continue
			}

		}
	}
}
