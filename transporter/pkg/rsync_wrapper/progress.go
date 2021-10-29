package rsync_wrapper

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"sync/atomic"
	"time"

	"transporter/pkg/client"
)

const (
	errNonNumeric                   = "ascii char is non-numeric"
	progressBufSize                 = 32

	// progressLine format: #xfr99
	progressLineFirstChar           = '#'
	progressNumStartIndex           = 4
	progressReporterIntervalDefault = 5
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

	log.Println("!![Info]already creat reader of stdout, start read loop..")

	for {
		select {
		case <-ctx.Done():
			log.Println("!![Info]get notify of progress cancel func")
			return

		default:

		}

		// use read method of reader, in this case, use read method of *File.
		l, isPrefix, err = r.ReadLine()
		if err != nil {
			log.Println("!![Warning]get err when read line of stdout, direct break, err:", err.Error())
			break
		}

		// means content's length of line at output is large than progress buffer size
		if isPrefix {
			log.Println("!![Warning]content's length of line at output is large than progress buffer size")
			continue
		}

		// read empty line
		if len(l) == 0 {
			log.Println("!![Warning]read empty of stdout")
			continue
		}

		// not progress line
		if l[0] != progressLineFirstChar {
			log.Println("!![Warning]first char is not #")
			continue
		}

		num, err = atoi(l[progressNumStartIndex:])
		if err != nil {
			log.Println("!![Warning]progress num is unavailable")
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

func reportProgress(ctx context.Context, progressNum *uint32, addr string, rc *client.ReportClient, reportInterval int) {
	var progressReporterInterval int
	if reportInterval <= 0 {
		progressReporterInterval = progressReporterIntervalDefault
	}

	log.Println("!![Info]report progress interval:", progressReporterInterval, "second")
	t := time.NewTicker(time.Duration(progressReporterInterval) * time.Second)

	var (
		currentProgressNum uint32
		reqContent         reqResult
	)

	log.Println("!![Info]ready to report progress, start loop..")

	for {
		select {
		case <-ctx.Done():
			t.Stop()
			log.Println("!![Info]get notify of progress cancel func, stop time ticker and return")
			return

		case <-t.C:
			currentProgressNum = atomic.LoadUint32(progressNum)

			reqContent.Count = int64(currentProgressNum)
			reqContentB, err := json.Marshal(&reqContent)
			if err != nil {
				log.Println("!![Warning] failed to marshal reqcontent of progress, err:", err.Error())
				continue
			}

			err = rc.Report(addr, client.ContentType, reqContentB)
			if err != nil {
				log.Println("!![Warning] failed to send http req of report, err:", err.Error())
				continue
			}

		}
	}
}
