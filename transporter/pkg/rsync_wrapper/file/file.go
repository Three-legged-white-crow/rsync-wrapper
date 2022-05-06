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
			log.Println("[CopyFile-Error]Retry limit reached, exit with ErrRetryLImit(208)")
			return exit_code.ErrRetryLimit
		}

		c := exec.Command(rsyncBinPath, cmdContent...)
		log.Println("[CopyFile-Info]Run command:", c.String(), "retry number:", currentRetryNum)

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
		log.Println(
			"[CopyFile-Error]Get subprocess exit error:", processExitErr.Error(),
			"exit code:", processExitErr.ExitCode(),
			"subprocess pid", processExitErr.Pid())

		finalExitCode, ok = rsync_wrapper.ExitCodeConvertWithStderr(string(stdoutStderr))
		if ok {
			log.Println("[CopyFile-Error]Matched stand file system exit code and exit:", finalExitCode)
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
		log.Println("[CopyFile-Warning]Get exit error but it is a recoverable error, will retry exec command")
		continue
	}

}
