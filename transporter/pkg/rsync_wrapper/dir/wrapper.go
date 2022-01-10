package dir

import (
	"bytes"
	"context"
	"log"
	"os/exec"
	"strings"

	"transporter/pkg/client"
	"transporter/pkg/exit_code"
	"transporter/pkg/rsync_wrapper"
)

const (
	retryMaxLimit       = 3
	rsyncBinPath        = "/usr/local/bin/rsync"
	rsyncOptionBasic    = "-rlptgoHA"
	rsyncOptionProgress = "--progress"
	rsyncOptionPartial  = "--partial"
	rsyncOptionSparse   = "--sparse"
	rsyncOptionFilter   = "--filter="
)

// reqResult is result of rsync cmd to report.
type reqResult struct {
	CurrentCount int64  `json:"current_count"` // currnet transfer file progress number
	TotalCount   int64  `json:"total_count"`   // total check file progress number
	Message      string `json:"message"`       // rsync stderr content
	ErrCode      int64  `json:"errcode"`       // exit code
	Reason       string `json:"reason"`        // reason of exit error
}

type ReqContent struct {
	SrcPath          string
	DestPath         string
	IsReportProgress bool
	IsReportStderr   bool
	IsHandleSparse   bool
	ReportClient     *client.ReportClient
	ReportInterval   int
	ReportAddr       string
	RetryLimit       int
	FilterList       []string
}

// Run run rsync command and if err return by rsync is recoverable will auto retry.
func Run(req ReqContent) int {
	var (
		res               resultRsync
		currentRetryNum   = 0
		currentRetryLimit int
		finalExitCode     int
		isExitDirect      bool = false
	)

	currentRetryLimit = req.RetryLimit
	if currentRetryLimit < 0 {
		currentRetryLimit = retryMaxLimit
	}
	log.Println("[copy-Info]Limit of retry dir copy:", currentRetryLimit)

	for {

		if currentRetryNum > currentRetryLimit {
			curExitCode := rsync_wrapper.ExitCodeConvert(res.exitCode)
			if req.IsReportStderr {
				_ = reportStderr(curExitCode, res.exitReason, res.stdErr, req.ReportAddr, req.ReportClient)
				_ = reportStderr(exit_code.ErrRetryLimit, exit_code.ErrMsgMaxLimitRetry, res.stdErr, req.ReportAddr, req.ReportClient)
			}
			log.Println(exit_code.ErrMsgMaxLimitRetry)
			log.Println("[Retry Limit]Latest process exit code:", res.exitCode, "latest retry count:", retryMaxLimit)
			return exit_code.ErrRetryLimit
		}

		res = runRsync(req)
		if res.exitCode == rsync_wrapper.ErrOK {
			log.Println(exit_code.ErrMsgSucceed)
			log.Println("[Complete]process exit code:", res.exitCode, "exit reason:", res.exitReason)
			return exit_code.Succeed
		}

		// if stderr of result is not nil, try get std exit code according std eror desc
		if len(res.stdErr) != 0 {
			log.Println("[copy-Warning]Stderr is not empty:", res.stdErr)

			// last exec, if exit with ErrVanished, retry command.
			// if rsync exited because file vanished, not count retry num.
			if res.exitCode == rsync_wrapper.ErrVanished {
				log.Println("[copy-Warning]File(s) vanished on sender side")
				continue
			}

			exitCodeStderr, ok := rsync_wrapper.ExitCodeConvertWithStderr(res.stdErr)
			if ok {
				log.Println("[copy-Error]Get exit code from stderr:", exitCodeStderr)
				finalExitCode = exitCodeStderr
				isExitDirect = true
			}
		} else {
			log.Println("[copy-Warning]Stderr is empty")
		}

		if isExitDirect {
			log.Println("[copy-Error]Process direct exit with code:", finalExitCode)
			return finalExitCode
		}

		if rsync_wrapper.IsErrUnRecoverable(res.exitCode) {
			curExitCode := rsync_wrapper.ExitCodeConvert(res.exitCode)
			if req.IsReportStderr {
				_ = reportStderr(curExitCode, res.exitReason, res.stdErr, req.ReportAddr, req.ReportClient)
			}
			log.Println(exit_code.ErrMsgUnrecoverable)
			log.Println("[Unrecoverable Err]process exit code:", res.exitCode, "exit reason:", res.exitReason, "stderr:", res.stdErr)
			return curExitCode
		}

		// last exec, get a recoverable error, retry command.
		// if rsync exited because file vanished, not count retry num.
		if res.exitCode == rsync_wrapper.ErrVanished {
			log.Println("[copy-Warning]File(s) vanished on sender side")
			continue
		}

		currentRetryNum += 1
		log.Println("[Retry]process count:", currentRetryNum)
		log.Println("[Retry]process exit code:", res.exitCode, "exit reason:", res.exitReason, "stderr:", res.stdErr)
		log.Println("[Retry]---------------------------------------------------------------------------------------")
	}
}

type resultRsync struct {
	exitCode   int
	exitReason string
	stdErr     string
}

// runRsync run rsync command and get stdout and stderr.
func runRsync(req ReqContent) (res resultRsync) {
	res.exitCode = rsync_wrapper.ErrOK
	res.exitReason = rsync_wrapper.ErrOKMsg

	var c *exec.Cmd
	var cmdArgList = []string{rsyncOptionBasic, rsyncOptionPartial}
	if req.IsHandleSparse {
		cmdArgList = append(cmdArgList, rsyncOptionSparse)
	}
	if req.IsReportProgress {
		cmdArgList = append(cmdArgList, rsyncOptionProgress)
	}

	var filterArg = strings.Builder{}
	var filterArgStr string
	if len(req.FilterList) > 0 {
		for _, rule := range req.FilterList {
			if len(rule) == 0 {
				continue
			}

			filterArg.Reset()
			filterArg.WriteString(rsyncOptionFilter)
			filterArg.WriteString(rule)
			filterArgStr = filterArg.String()
			cmdArgList = append(cmdArgList, filterArgStr)
		}
	}

	cmdArgList = append(cmdArgList, req.SrcPath)
	cmdArgList = append(cmdArgList, req.DestPath)

	c = exec.Command(rsyncBinPath, cmdArgList...)
	log.Println("[copy-Info]cmd string:", c.String())

	if req.IsReportProgress {
		log.Println("[copy-Info]read stdout of cmd turn on")

		stdoutPipe, err := c.StdoutPipe()
		if err != nil {
			res.exitCode = rsync_wrapper.ErrCreatePipe
			res.exitReason = err.Error()
			return res
		}

		ctx, cancelProgressFunc := context.WithCancel(context.Background())
		defer cancelProgressFunc()

		var curProgressNum, totalProgressNum uint32
		go readStdout(ctx, stdoutPipe, &curProgressNum, &totalProgressNum)
		go reportProgress(ctx, &curProgressNum, &totalProgressNum, req.ReportAddr, req.ReportClient, req.ReportInterval)

	}

	// because of get exit code from stderr, so always read stderr
	log.Println("[copy-Info]read stderr of cmd turn on")
	var stderrBuf bytes.Buffer
	c.Stderr = &stderrBuf

	defer func() {
		res.stdErr = stderrBuf.String()
	}()

	errStart := c.Start()
	if errStart != nil {
		res.exitCode = rsync_wrapper.ErrStartCmd
		res.exitReason = errStart.Error()
		return res
	}

	log.Println("[copy-Info]succeed to start command")

	errWait := c.Wait()
	if errWait != nil {
		log.Println("[copy-Warning]get wait cmd err:", errWait.Error())

		if rsync_wrapper.IsWaitProcessErr(errWait) {
			res.exitCode = rsync_wrapper.ErrWaitProcess
			res.exitReason = errWait.Error()
			return res
		}

		processExitErr, ok := errWait.(*exec.ExitError)
		if ok {
			res.exitCode = processExitErr.ExitCode()
			res.exitReason = errWait.Error()
			return res
		}

		log.Println("[copy-Warning]get wait err, but err is not exiterr:", errWait.Error())
		res.exitCode = rsync_wrapper.ErrWaitProcess
		res.exitReason = errWait.Error()
		return res
	}

	return res
}
