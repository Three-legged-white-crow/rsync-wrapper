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


type ReqFileList struct {
	SrcMountPath          string
	DestMountPath         string
	InputRecordFile       string
	OutputRecordFile      string
	IgnoreSrcNotExist     bool
	IgnoreSrcIsDir        bool
	IgnoreDstIsExistDir   bool
	OverWriteDstExistFile bool
}

func CopyFileList(req ReqFileList) {
	// todo: load src file rel path and dest file rel path from inFile

	// todo: build complete src file path and dest file path

	// todo: check src / dest exist and is dir

	// todo: call Run to rsync file

	// todo: record failed to outFile

}