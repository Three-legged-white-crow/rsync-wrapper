package file

import (
	"errors"
	"log"
	"os/exec"

	"golang.org/x/sys/unix"
	"transporter/pkg/exit_code"
	"transporter/pkg/rsync_wrapper"
)

const (
	rsyncBinPath       = "/usr/local/bin/rsync"
	rsyncOptionBasic   = "-rlptgoHA"
	rsyncOptionPartial = "--partial"
	rsyncOptionSparse  = "--sparse"
	retryMaxLimit      = 3
)

type ReqContent struct {
	SrcPath        string
	DestPath       string
	IsHandleSparse bool
	RetryLimit     int
}

func CopyFile(req ReqContent) int {

	var (
		finalExitCode     int
		ok                bool
		currentRetryLimit int
		currentRetryNum   int
		stdoutStderr      []byte
		err               error
	)

	currentRetryLimit = req.RetryLimit
	if currentRetryLimit < 0 {
		currentRetryLimit = retryMaxLimit
	}

	cmdContent := []string{rsyncOptionBasic, rsyncOptionPartial}
	if req.IsHandleSparse {
		cmdContent = append(cmdContent, rsyncOptionSparse)
	}

	cmdContent = append(cmdContent, req.SrcPath)
	cmdContent = append(cmdContent, req.DestPath)

	for {
		if currentRetryNum > currentRetryLimit {
			return exit_code.ErrRetryLimit
		}

		c := exec.Command(rsyncBinPath, cmdContent...)

		stdoutStderr, err = c.CombinedOutput()
		if err == nil {
			return exit_code.Succeed
		}

		if errors.Is(err, unix.EINVAL) {
			return exit_code.ErrInvalidArgument
		}

		var processExitErr *exec.ExitError
		processExitErr, ok = err.(*exec.ExitError)
		if !ok {
			log.Println(
				"[CopyFile-Error]Get err but not ExitError when copy src:", req.SrcPath,
				"to dest:", req.DestPath,
				"and err:", err.Error())
			return exit_code.ErrSystem
		}

		finalExitCode, ok = rsync_wrapper.ExitCodeConvertWithStderr(string(stdoutStderr))
		if ok {
			return finalExitCode
		}

		processExitCode := processExitErr.ExitCode()
		if rsync_wrapper.IsErrUnRecoverable(processExitCode) {
			log.Println(
				"[CopyFile-Error]Get unrecoverable err when copy src:", req.SrcPath,
				"to dest:", req.DestPath,
				"and err:", processExitErr.Error())
			finalExitCode = rsync_wrapper.ExitCodeConvert(processExitCode)
			return finalExitCode
		}

		currentRetryNum += 1
		continue
	}

}
