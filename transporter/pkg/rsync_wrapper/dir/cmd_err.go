package dir

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"strings"

	"transporter/pkg/client"
)

func readStderr(ctx context.Context, reader io.Reader, processStdErrChan chan<- string) {

	var (
		cmdStdErrContent = strings.Builder{}
		l                string
		err              error
	)

	defer func() {
		processStdErrChan <- cmdStdErrContent.String()
		close(processStdErrChan)
	}()

	r := bufio.NewReader(reader)

	log.Println("[copy-Info]already create stderr reader, start loop..")
	for {
		select {
		case <-ctx.Done():
			log.Println("[copy-Info]get notify of stderr cancel func")
			return

		default:

		}

		l, err = r.ReadString('\n')
		if err != nil {
			log.Println("[copy-Warning]get err when read stderr content, direct break, err:", err.Error())
			break
		}

		cmdStdErrContent.WriteString(l)
	}
}

func reportStderr(exitCode int, exitReason, stdErr, addr string, rc *client.ReportClient) error {
	reqContent := reqResult{
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
