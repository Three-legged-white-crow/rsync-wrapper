package dir

import (
	"bytes"
	"context"
	"log"
	"os/exec"

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
)

// reqResult is result of rsync cmd to report.
type reqResult struct {
	CurrentCount int64  `json:"current_count"` // currnet transfer file progress number
	TotalCount   int64  `json:"total_count"`   // total check file progress number
	Message      string `json:"message"`       // rsync stderr content
	ErrCode      int64  `json:"errcode"`       // exit code
	Reason       string `json:"reason"`        // reason of exit error
}

type ReqRun struct {
	SrcPath          string
	DestPath         string
	IsReportProgress bool
	IsReportStderr   bool
	ReportClient     *client.ReportClient
	ReportInterval   int
	ReportAddr       string
}

// Run run rsync command and if err return by rsync is recoverable will auto retry.
func Run(req ReqRun) int {
	var (
		res             resultRsync
		currentRetryNum = 0
		finalExitCode   int
		isExitDirect    bool = false
	)

	for {

		if currentRetryNum >= retryMaxLimit {
			curExitCode := rsync_wrapper.ExitCodeConvert(res.exitCode)
			if req.IsReportStderr {
				_ = reportStderr(curExitCode, res.exitReason, res.stdErr, req.ReportAddr, req.ReportClient)
			}
			log.Println(exit_code.ErrMsgMaxLimitRetry)
			log.Println("[Retry Limit]Use latest process exit code:", res.exitCode, "latest retry count:", retryMaxLimit)
			return curExitCode
		}

		res = runRsync(req.SrcPath, req.DestPath, req.ReportAddr, req.IsReportProgress, req.ReportClient, req.ReportInterval)
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
		}else {
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
func runRsync(src, dest, addr string, isReportProgress bool, rc *client.ReportClient, reportInterval int) (res resultRsync) {
	res.exitCode = rsync_wrapper.ErrOK
	res.exitReason = rsync_wrapper.ErrOKMsg

	var c *exec.Cmd

	if isReportProgress {
		log.Println("[copy-Info]read stdout of cmd turn on")

		c = exec.Command(
			rsyncBinPath,
			rsyncOptionBasic,
			rsyncOptionPartial,
			rsyncOptionProgress,
			src,
			dest)

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
		go reportProgress(ctx, &curProgressNum, &totalProgressNum, addr, rc, reportInterval)

	} else {
		c = exec.Command(
			rsyncBinPath,
			rsyncOptionBasic,
			rsyncOptionPartial,
			src,
			dest)
	}

	log.Println("[copy-Info]cmd string:", c.String())

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
