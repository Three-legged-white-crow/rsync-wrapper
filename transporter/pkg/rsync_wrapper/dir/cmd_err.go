package dir

import (
	"encoding/json"

	"transporter/pkg/client"
)

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
