package rsync_wrapper

import (
	"bufio"
	"bytes"
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

	// progressLine format: 99-200
	splitSymbol                     = '-'
	progressReporterIntervalDefault = 5
)

// readStdout read content of stdout with pipe, parsed progress.
func readStdout(ctx context.Context, reader io.Reader, curProgressNum, totalProgressNum *uint32) {
	var (
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

		progressParse(l, curProgressNum, totalProgressNum)
	}
}

func progressParse(progressInfo []byte, curProgressNum, totalProgressNum *uint32) {
	splitIndex := bytes.IndexByte(progressInfo, splitSymbol)
	curFileNum, err := atoi(progressInfo[:splitIndex])
	if err != nil {
		log.Println("!![Warning]current progress num is unavailable")
		return
	}

	if splitIndex + 1 == len(progressInfo) {
		log.Println("!![Warning]no total progress num")
		return
	}

	totalFileNum, err := atoi(progressInfo[splitIndex+1:])
	if err != nil {
		log.Println("!![Warning]total progress num is unavailable")
		return
	}

	atomic.StoreUint32(curProgressNum, curFileNum)
	atomic.StoreUint32(totalProgressNum, totalFileNum)
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

func reportProgress(ctx context.Context, curProgressNum, totalProgressNum *uint32, addr string, rc *client.ReportClient, reportInterval int) {
	var progressReporterInterval int
	if reportInterval <= 0 {
		progressReporterInterval = progressReporterIntervalDefault
	}

	log.Println("!![Info]report progress interval:", progressReporterInterval, "second")
	t := time.NewTicker(time.Duration(progressReporterInterval) * time.Second)

	var (
		currentFileNum uint32
		totalFileNum   uint32
		reqContent     reqResult
	)

	log.Println("!![Info]ready to report progress, start loop..")

	for {
		select {
		case <-ctx.Done():
			t.Stop()
			log.Println("!![Info]get notify of progress cancel func, stop time ticker and return")
			return

		case <-t.C:
			currentFileNum = atomic.LoadUint32(curProgressNum)
			totalFileNum = atomic.LoadUint32(totalProgressNum)

			reqContent.CurrentCount = int64(currentFileNum)
			reqContent.TotalCount = int64(totalFileNum)
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
