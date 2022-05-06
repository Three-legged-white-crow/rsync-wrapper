//go:build amd64 && linux
// +build amd64,linux

package rsync_wrapper

import (
	"errors"
	"log"
	"os"
	"strings"

	"golang.org/x/sys/unix"
	"transporter/pkg/exit_code"
)

const (
	ErrOK    = 0
	ErrOKMsg = "process succeed complete with exit code 0"

	// Error codes returned by rsync.
	ErrSyntax      = 1   // syntax or usage error
	ErrProtocol    = 2   // protocol incompatibility
	ErrFileselect  = 3   // errors selecting input/output files, dirs
	ErrUnsupported = 4   // requested action not supported
	ErrStartclient = 5   // error starting client-server protocol
	ErrSocketio    = 10  // error in socket IO
	ErrFileio      = 11  // error in file IO
	ErrStreamio    = 12  // error in rsync protocol data stream
	ErrMessageio   = 13  // errors with program diagnostics
	ErrIPC         = 14  // error in IPC code
	ErrCrashed     = 15  // sibling process crashed
	ErrTerminated  = 16  // sibling process terminated abnormally
	ErrSignal1     = 19  // received SIGUSR1
	ErrSignal      = 20  // received SIGINT, SIGTERM, or SIGHUP. SIGKILL will handle as unrecoverable err
	ErrWaitChild   = 21  // some error returned by waitpid()
	ErrMalloc      = 22  // error allocating core memory buffers
	ErrPartial     = 23  // partial transfer, some files/attrs were not transferred (see previous errors
	ErrVanished    = 24  // file(s) vanished on sender side, some files vanished before they could be transferred
	ErrDelLimit    = 25  // skipped some deletes due to --max-delete
	ErrTimeout     = 30  // timeout in data send/receive
	ErrConTimeout  = 35  // timeout waiting for daemon connection
	ErrCmdFailed   = 124 // remote shell failed
	ErrCmdKilled   = 125 // remote shell killed
	ErrCmdRun      = 126 // remote command could not be run
	ErrCmdNotfound = 127 // remote command not found

	// Error codes when start rsync command with exec
	ErrCreatePipe  = -10 // failed to create pipe for stdin/stdout/stderr
	ErrStartCmd    = -11 // failed to start the specified command
	ErrWaitProcess = -12 // failed to wait specified command process
)

var (
	// if get recoverable err of rsync, rsync wrapper will retry run rsync command
	recoverableErrList = [23]int{
		ErrSocketio,
		ErrFileio,
		ErrStreamio,
		ErrMessageio,
		ErrIPC,
		ErrCrashed,
		ErrTerminated,
		ErrSignal1,
		ErrSignal,
		ErrWaitChild,
		ErrMalloc,
		ErrPartial,
		ErrVanished,
		ErrDelLimit,
		ErrTimeout,
		ErrConTimeout,
		ErrCmdFailed,
		ErrCmdKilled,
		ErrCmdRun,
		ErrCmdNotfound,
		ErrCreatePipe,
		ErrStartCmd,
		ErrWaitProcess,
	}

	// if get unRecoverable err of rsync, rsync wrapper will exit direct,
	// also includes other errors that are unrecoverable and will cause the process to terminate.
	unRecoverableErrList = [5]int{
		ErrSyntax,
		ErrProtocol,
		ErrFileselect,
		ErrUnsupported,
		ErrStartclient,

		// other errors like:
		// errSIGKILL
		// errSIGBUS
		// errSIGSEGV
		// ......

	}

	rsyncExitCodeMap = map[int]int{
		ErrOK:          exit_code.Succeed,
		ErrSyntax:      exit_code.ErrInvalidArgument,
		ErrProtocol:    exit_code.ErrSystem,
		ErrFileselect:  exit_code.ErrNoSuchFileOrDir,
		ErrUnsupported: exit_code.ErrInvalidArgument,
		ErrStartclient: exit_code.ErrSystem,
		ErrSocketio:    exit_code.ErrIOError,
		ErrFileio:      exit_code.ErrIOError,
		ErrStreamio:    exit_code.ErrIOError,
		ErrMessageio:   exit_code.ErrIOError,
		ErrIPC:         exit_code.ErrSystem,
		ErrCrashed:     exit_code.ErrSystem,
		ErrTerminated:  exit_code.ErrSystem,
		ErrSignal1:     exit_code.ErrSystem,
		ErrSignal:      exit_code.ErrSystem,
		ErrWaitChild:   exit_code.ErrSystem,
		ErrMalloc:      exit_code.ErrSystem,
		ErrPartial:     exit_code.ErrPermissionDenied,
		ErrDelLimit:    exit_code.ErrSystem,
		ErrTimeout:     exit_code.ErrSystem,
		ErrConTimeout:  exit_code.ErrSystem,
		ErrCmdFailed:   exit_code.ErrSystem,
		ErrCmdKilled:   exit_code.ErrSystem,
		ErrCmdRun:      exit_code.ErrSystem,
		ErrCmdNotfound: exit_code.ErrSystem,
		ErrCreatePipe:  exit_code.ErrSystem,
		ErrStartCmd:    exit_code.ErrSystem,
		ErrWaitProcess: exit_code.ErrSystem,
	}

	// if get one of these err msg, wrapper should exit directly
	stdExitCodeMsgList = [12]string{
		exit_code.ErrMsgNOENT,
		exit_code.ErrMsgIOError,
		exit_code.ErrMsgPermissionDenided,
		exit_code.ErrMsgDeviceBusy,
		exit_code.ErrMsgFileIsExists,
		exit_code.ErrMsgNotDirectory,
		exit_code.ErrMsgIsDirectory,
		exit_code.ErrMsgInval,
		exit_code.ErrMsgNoSpace,
		exit_code.ErrMsgFSReadOnly,
		exit_code.ErrMsgDiskQuota,
		exit_code.ErrMsgFileStale,
	}

	stdExitCodeMap = map[string]int{
		exit_code.ErrMsgNOENT:             exit_code.ErrNoSuchFileOrDir,
		exit_code.ErrMsgIOError:           exit_code.ErrIOError,
		exit_code.ErrMsgPermissionDenided: exit_code.ErrPermissionDenied,
		exit_code.ErrMsgDeviceBusy:        exit_code.ErrDeviceIsBusy,
		exit_code.ErrMsgFileIsExists:      exit_code.ErrFileIsExists,
		exit_code.ErrMsgNotDirectory:      exit_code.ErrNotDirectory,
		exit_code.ErrMsgIsDirectory:       exit_code.ErrIsDirectory,
		exit_code.ErrMsgInval:             exit_code.ErrInvalidArgument,
		exit_code.ErrMsgNoSpace:           exit_code.ErrNoSpaceLeftOnDevice,
		exit_code.ErrMsgFSReadOnly:        exit_code.ErrFileSystemIsReadOnly,
		exit_code.ErrMsgDiskQuota:         exit_code.ErrDiskQuota,
		exit_code.ErrMsgFileStale:         exit_code.ErrFileStale,
	}
)

// isErrRecoverable return true if err is recoverable.
func isErrRecoverable(errCode int) bool {
	for _, ec := range recoverableErrList {
		if ec == errCode {
			return true
		}
	}

	return false
}

// IsErrUnRecoverable return true if err is unrecoverable.
func IsErrUnRecoverable(errCode int) bool {
	for _, ec := range unRecoverableErrList {
		if ec == errCode {
			return true
		}
	}

	return false
}

// IsWaitProcessErr return true if err return by Wait method of Process.
func IsWaitProcessErr(err error) bool {
	if errors.Is(err, unix.EINVAL) {
		return true
	}

	_, ok := err.(*os.SyscallError)
	return ok
}

func ExitCodeConvert(errCode int) int {
	exitCode, ok := rsyncExitCodeMap[errCode]
	if !ok {
		return exit_code.ErrSystem
	}

	return exitCode
}

func ExitCodeConvertWithStderr(errContent string) (int, bool) {
	if len(errContent) == 0 {
		log.Println("[Match-FS-ExitCode]Combined output is empty, end match")
		return 0, false
	}

	log.Println("[Match-FS-ExitCode]Combined output:", errContent)
	errStrList := strings.Split(errContent, "\n")

	errStrLen := len(errStrList)
	var curErrStr string
	for i := errStrLen - 1; i >= 0; i -= 1 {
		curErrStr = errStrList[i]

		for _, msg := range stdExitCodeMsgList {
			if strings.Contains(curErrStr, msg) {
				exitCode := stdExitCodeMap[msg]
				log.Println(
					"[Match-FS-ExitCode]Succeed to match fs errMsg:", msg,
					"output line num:", i,
					"output line content:", curErrStr,
					"fs exit code:", exitCode)
				return exitCode, true
			}
		}
	}

	log.Println("[Match-FS-ExitCode]All combined output has been match with fs errmsg, nothing matched")
	return 0, false
}
