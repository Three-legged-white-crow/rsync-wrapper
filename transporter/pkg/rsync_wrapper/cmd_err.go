package rsync_wrapper

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"

	"transporter/pkg/client"
)

func readStderr(ctx context.Context, reader io.Reader, processStdErrChan chan<- string) {

	var (
		cmdStdErrContent = strings.Builder{}
		l                string
		err              error
	)

	r := bufio.NewReader(reader)

	for {
		select {
		case <-ctx.Done():
			processStdErrChan <- cmdStdErrContent.String()
			close(processStdErrChan)

		default:

		}

		l, err = r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			continue
		}

		cmdStdErrContent.WriteString(l)
	}
}

func reportStderr(exitCode int, exitReason, stdErr, addr string, rc *client.ReportClient) error {
	reqContent := reqResult{
		Count:   0,
		Message: stdErr,
		ErrCode: int64(exitCode),
		Reason:  exitReason,
	}

	reqContentB, err := json.Marshal(&reqContent)
	if err != nil {
		return err
	}

	err = rc.Report(addr, client.ContentType, reqContentB)
	if err != nil {
		return err
	}

	return nil
}
