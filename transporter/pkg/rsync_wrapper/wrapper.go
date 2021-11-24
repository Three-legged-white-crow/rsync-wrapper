package rsync_wrapper

import (
	"context"
	"log"
	"os/exec"

	"transporter/pkg/client"
	"transporter/pkg/exit_code"
)

const (
	retryMaxLimit       = 3
	rsyncBinPath        = "/usr/local/bin/rsync"
	rsyncOptionBasic    = "-rlptgoH"
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

// Run run rsync command and if err return by rsync is recoverable will auto retry.
func Run(src, dest, addr string, isReportProgress, isReportStderr bool, rc *client.ReportClient, reportInterval int) int {
	var (
		res             resultRsync
		currentRetryNum = 0
	)

	for {

		if currentRetryNum == retryMaxLimit {
			curExitCode := exitCodeConvert(res.exitCode)
			if isReportStderr {
				_ = reportStderr(curExitCode, res.exitReason, res.stdErr, addr, rc)
			}
			log.Println(exit_code.ErrMsgMaxLimitRetry)
			log.Println("[Retry Limit]Use latest process exit code:", res.exitCode, "latest retry count:", retryMaxLimit)
			return curExitCode
		}

		res = runRsync(src, dest, addr, isReportProgress, isReportStderr, rc, reportInterval)
		if res.exitCode == errOK {
			log.Println(exit_code.ErrMsgSucceed)
			log.Println("[Complete]process exit code:", res.exitCode, "exit reason:", res.exitReason)
			return exit_code.Succeed
		}

		if isErrUnRecoverable(res.exitCode) {
			curExitCode := exitCodeConvert(res.exitCode)
			if isReportStderr {
				_ = reportStderr(curExitCode, res.exitReason, res.stdErr, addr, rc)
			}
			log.Println(exit_code.ErrMsgUnrecoverable)
			log.Println("[Unrecoverable Err]process exit code:", res.exitCode, "exit reason:", res.exitReason, "stderr:", res.stdErr)
			return curExitCode
		}

		// last exec, get a recoverable error, retry command.
		// if rsync exited because file vanished, not count retry num.
		if res.exitCode == errVanished {
			log.Println("!![Warning]File(s) vanished on sender side")
			continue
		}

		currentRetryNum += 1
		log.Println("[Retry]process count:", currentRetryNum)
		log.Println("[Retry]process exit code:", res.exitCode, "exit reason:", res.exitReason, "stderr:", res.stdErr)
	}
}

type resultRsync struct {
	exitCode   int
	exitReason string
	stdErr     string
}

// runRsync run rsync command and get stdout and stderr.
func runRsync(src, dest, addr string, isReportProgress, isReportStderr bool, rc *client.ReportClient, reportInterval int) resultRsync {
	var res = resultRsync{
		exitCode:   errOK,
		exitReason: errOKMsg,
		stdErr:     "",
	}

	var c *exec.Cmd

	if isReportProgress {
		log.Println("!![Info]read stdout of cmd turn on")

		c = exec.Command(
			rsyncBinPath,
			rsyncOptionBasic,
			rsyncOptionPartial,
			rsyncOptionProgress,
			src,
			dest)

		stdoutPipe, err := c.StdoutPipe()
		if err != nil {
			res.exitCode = errCreatePipe
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

	log.Println("!![Info]cmd string:", c.String())

	if isReportStderr {
		log.Println("!![Info]read stderr of cmd turn on")

		stderrPipe, err := c.StderrPipe()
		if err != nil {
			res.exitCode = errCreatePipe
			res.exitReason = err.Error()
			return res
		}

		// collect stderr content of specified process
		processStdErrChan := make(chan string, 1)
		defer func(c <-chan string) {
			stdErrInfo := <-c
			res.stdErr = stdErrInfo
			log.Println("!![Info]succeed to get stderrinfo from stderr reader")
		}(processStdErrChan)

		ctx, cancelStderrFunc := context.WithCancel(context.Background())
		defer cancelStderrFunc()

		go readStderr(ctx, stderrPipe, processStdErrChan)
	}

	errStart := c.Start()
	if errStart != nil {
		res.exitCode = errStartCmd
		res.exitReason = errStart.Error()
		return res
	}

	log.Println("!![Info]succeed to start command")

	errWait := c.Wait()
	if errWait != nil {
		log.Println("!![Warning]get wait cmd err:", errWait.Error())

		if isWaitProcessErr(errWait) {
			res.exitCode = errWaitProcess
			res.exitReason = errWait.Error()
			return res
		}

		processExitErr, ok := errWait.(*exec.ExitError)
		if ok {
			res.exitCode = processExitErr.ExitCode()
			res.exitReason = errWait.Error()
			return res
		}

		log.Println("!![Warning]get wait err, but err is not exiterr:", errWait.Error())
		return res
	}

	return res
}
