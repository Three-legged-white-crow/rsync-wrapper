package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/report", reporter)
	err := http.ListenAndServe("0.0.0.0:9001", nil)
	if err != nil {
		fmt.Println("get err:", err)
		os.Exit(1)
	}
}

type reqResult struct {
	CurrentCount int64  `json:"current_count"` // currnet transfer file progress number
	TotalCount   int64  `json:"total_count"`   // total check file progress number
	Message      string `json:"message"`       // rsync stderr content
	ErrCode      int64  `json:"errcode"`       // exit code
	Reason       string `json:"reason"`        // reason of exit error
}

func reporter(w http.ResponseWriter, r *http.Request) {
	reqContentB, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	reqContent := reqResult{}
	err = json.Unmarshal(reqContentB, &reqContent)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	if reqContent.CurrentCount > 0 && reqContent.TotalCount > 0 {
		log.Println("progress current num:", reqContent.CurrentCount, "total num:", reqContent.TotalCount)
	}

	if len(reqContent.Message) > 0 {
		log.Println("stderr msg:", reqContent.Message)
	}

	if reqContent.ErrCode != 0 {
		log.Println("err exit code:", reqContent.ErrCode)
	}

	if len(reqContent.Reason) > 0 {
		log.Println("err reason:", reqContent.Reason)
	}

	w.WriteHeader(http.StatusOK)
}
