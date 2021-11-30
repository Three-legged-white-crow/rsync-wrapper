package file

import (
	"errors"
	"os/exec"

	"golang.org/x/sys/unix"
	"transporter/pkg/exit_code"
	"transporter/pkg/rsync_wrapper"
)

const (
	rsyncBinPath       = "/usr/local/bin/rsync"
	rsyncOptionBasic   = "-rlptgoHA"
	rsyncOptionPartial = "--partial"
)


func CopyFile(src, dest string) int {

	c := exec.Command(
		rsyncBinPath,
		rsyncOptionBasic,
		rsyncOptionPartial,
		src,
		dest)

	var (
		finalExitCode int
		ok            bool
	)
	stdoutStderr, err := c.CombinedOutput()
	if err != nil {
		if errors.Is(err, unix.EINVAL) {
			return exit_code.ErrInvalidArgument
		}

		var processExitErr *exec.ExitError
		processExitErr, ok = err.(*exec.ExitError)
		if ok {
			finalExitCode, ok = rsync_wrapper.ExitCodeConvertWithStderr(string(stdoutStderr))
			if !ok {
				finalExitCode = rsync_wrapper.ExitCodeConvert(processExitErr.ExitCode())
			}

			return finalExitCode
		}

		return exit_code.ErrSystem
	}

	finalExitCode, ok = rsync_wrapper.ExitCodeConvertWithStderr(string(stdoutStderr))
	if ok {
		return finalExitCode
	}

	return exit_code.Succeed
}