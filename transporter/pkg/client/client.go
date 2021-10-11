package client

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"transporter/pkg/exit_code"
)

const (
	timeOutReport = 30 // unit is second
	ContentType   = "application/json"
)

// CheckAddr check is the specified address available.
func CheckAddr(addr string) bool {
	// at current, no check
	return true
}

// CheckAddrList check report addr is available.
func CheckAddrList(addrList ...string) bool {
	for _, addr := range addrList {
		isaddrReportProgressAvailable := CheckAddr(addr)
		if isaddrReportProgressAvailable {
			continue
		}

		log.Println(exit_code.ErrMsgReportAddr)
		log.Println("report addr:", addr)
		return false
	}

	return true
}

// ReportClient is use for report data to reportAddr
type ReportClient struct {
	HttpClient *http.Client
}

// NewReportClient retrun a new reportClient.
func NewReportClient() *ReportClient {
	hc := &http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       timeOutReport * time.Second,
	}
	return &ReportClient{HttpClient: hc}
}

// Report request reportAddr to report data.
func (rc *ReportClient) Report(reportAddr, contentType string, data []byte) error {
	dataReader := bytes.NewReader(data)
	resp, err := rc.HttpClient.Post(reportAddr, contentType, dataReader)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}
