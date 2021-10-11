package rsync_wrapper

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"os/exec"

	"transporter/pkg/client"
	"transporter/pkg/exit_code"
)

const (
	retryMaxLimit       = 3
	errNoFileOrDirStr   = "no such file or directory"
	ErrPathNotDir       = "path is not directory"
	permDir             = 0775
	rsyncBinPath        = "/usr/bin/rsync"
	rsyncOptionBasic    = "-rlptgo"
	rsyncOptionProgress = "--progress"
	rsyncOptionPartial  = "--partial"
)

// reqResult is result of rsync cmd to report.
type reqResult struct {
	Count   int64  `json:"count"`   // progress number
	Message string `json:"message"` // rsync stderr content
	ErrCode int64  `json:"errcode"` // exit code
	Reason  string `json:"reason"`  // reason of exit error
}

// Run run rsync command and if err return by rsync is recoverable will auto retry.
func Run(src, dest, addr string, isReportProgress, isReportStderr bool, rc *client.ReportClient) int {
	var (
		res             resultRsync
		currentRetryNum = 0
	)

	for {

		if currentRetryNum == retryMaxLimit {
			if isReportStderr {
				_ = reportStderr(res.exitCode, res.exitReason, res.stdErr, addr, rc)
			}
			log.Println(exit_code.ErrMsgMaxLimitRetry)
			return exit_code.ErrMaxLimitRetry
		}

		res = runRsync(src, dest, addr, isReportProgress, isReportStderr, rc)
		if res.exitCode == errOK {
			log.Println(exit_code.ErrMsgSucceed)
			log.Println("[Complete]exit code:", res.exitCode, "exit reason:", res.exitReason)
			return exit_code.Succeed
		}

		if isErrUnRecoverable(res.exitCode) {
			if isReportStderr {
				_ = reportStderr(res.exitCode, res.exitReason, res.stdErr, addr, rc)
			}
			log.Println(exit_code.ErrMsgUnrecoverable)
			log.Println("[Unrecoverable Err]exit code:", res.exitCode, "exit reason:", res.exitReason, "stderr:", res.stdErr)
			return exit_code.ErrUnrecoverable
		}

		// last exec, get a recoverable error, retry command
		currentRetryNum += 1
		log.Println("[retry]process count:", currentRetryNum)
		log.Println("[retry]process exit code:", res.exitCode, "exit reason:", res.exitReason, "stderr:", res.stdErr)
	}
}

type resultRsync struct {
	exitCode   int
	exitReason string
	stdErr     string
}

// runRsync run rsync command and get stdout and stderr.
func runRsync(src, dest, addr string, isReportProgress, isReportStderr bool, rc *client.ReportClient) resultRsync {
	var res = resultRsync{
		exitCode:   errOK,
		exitReason: errOKMsg,
		stdErr:     "",
	}

	var c *exec.Cmd
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	if isReportProgress {
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

		var progressNum uint32
		go readStdout(ctx, stdoutPipe, &progressNum)
		go reportProgress(ctx, &progressNum, addr, rc)

	} else {
		c = exec.Command(
			rsyncBinPath,
			rsyncOptionBasic,
			rsyncOptionPartial,
			src,
			dest)
	}

	if isReportStderr {
		stderrPipe, err := c.StderrPipe()
		if err != nil {
			res.exitCode = errCreatePipe
			res.exitReason = err.Error()
			return res
		}

		// collect stderr content of specified process
		processStdErrChan := make(chan string)
		defer func(c <-chan string) {
			stdErrInfo := <-c
			res.stdErr = stdErrInfo
		}(processStdErrChan)

		go readStderr(ctx, stderrPipe, processStdErrChan)
	}

	errStart := c.Start()
	if errStart != nil {
		res.exitCode = errStartCmd
		res.exitReason = errStart.Error()
		cancelFunc()
		return res
	}

	errWait := c.Wait()
	if errWait != nil {
		if isWaitProcessErr(errWait) {
			res.exitCode = errWaitProcess
			res.exitReason = errWait.Error()
			cancelFunc()
			return res
		}

		processExitErr, ok := errStart.(*exec.ExitError)
		if ok {
			res.exitCode = processExitErr.ExitCode()
			res.exitReason = errWait.Error()
			cancelFunc()
			return res
		}

		cancelFunc()
		return res
	}

	cancelFunc()
	return res
}

// CheckOrCreateDir check path is a dir, if not exist, create dir.
// If path is a exist dir or create a new dir according path, return nil.
func CheckOrCreateDir(dirPath string) error {
	f, err := os.Open(dirPath)
	if err != nil {
		errStr := err.(*fs.PathError).Unwrap().Error()
		if errStr == errNoFileOrDirStr {
			err = os.MkdirAll(dirPath, permDir)
			if err != nil {
				return err
			}

			return nil
		}

		return err
	}

	fInfo, err := f.Stat()
	if err != nil {
		return err
	}

	if !fInfo.IsDir() {
		return errors.New(ErrPathNotDir)
	}

	return nil
}
