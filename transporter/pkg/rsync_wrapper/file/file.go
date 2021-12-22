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
	retryMaxLimit      = 3
)

func CopyFile(src, dest string, retryLimt int) int {

	c := exec.Command(
		rsyncBinPath,
		rsyncOptionBasic,
		rsyncOptionPartial,
		src,
		dest)

	var (
		finalExitCode     int
		ok                bool
		currentRetryLimit int
		currentRetryNum   int
		stdoutStderr      []byte
		err               error
	)

	currentRetryLimit = retryLimt
	if currentRetryLimit < 0 {
		currentRetryLimit = retryMaxLimit
	}

	for {
		if currentRetryNum > currentRetryLimit {
			return exit_code.ErrRetryLimit
		}

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
				"[CopyFile-Error]Get err but not ExitError when copy src:", src,
				"to dest:", dest,
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
				"[CopyFile-Error]Get unrecoverable err when copy src:", src,
				"to dest:", dest,
				"and err:", processExitErr.Error())
			finalExitCode = rsync_wrapper.ExitCodeConvert(processExitCode)
			return finalExitCode
		}

		currentRetryNum += 1
		continue
	}

}
