package rsync_wrapper

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
	"transporter/pkg/exit_code"
)

const (
	errOK    = 0
	errOKMsg = "process succeed complete with exit code 0"

	// Error codes returned by rsync.
	errSyntax      = 1   // syntax or usage error
	errProtocol    = 2   // protocol incompatibility
	errFileselect  = 3   // errors selecting input/output files, dirs
	errUnsupported = 4   // requested action not supported
	errStartclient = 5   // error starting client-server protocol
	errSocketio    = 10  // error in socket IO
	errFileio      = 11  // error in file IO
	errStreamio    = 12  // error in rsync protocol data stream
	errMessageio   = 13  // errors with program diagnostics
	errIPC         = 14  // error in IPC code
	errCrashed     = 15  // sibling process crashed
	errTerminated  = 16  // sibling process terminated abnormally
	errSignal1     = 19  // received SIGUSR1
	errSignal      = 20  // received SIGINT, SIGTERM, or SIGHUP. SIGKILL will handle as unrecoverable err
	errWaitChild   = 21  // some error returned by waitpid()
	errMalloc      = 22  // error allocating core memory buffers
	errPartial     = 23  // partial transfer, some files/attrs were not transferred (see previous errors
	errVanished    = 24  // file(s) vanished on sender side, some files vanished before they could be transferred
	errDelLimit    = 25  // skipped some deletes due to --max-delete
	errTimeout     = 30  // timeout in data send/receive
	errConTimeout  = 35  // timeout waiting for daemon connection
	errCmdFailed   = 124 // remote shell failed
	errCmdKilled   = 125 // remote shell killed
	errCmdRun      = 126 // remote command could not be run
	errCmdNotfound = 127 // remote command not found

	// Error codes when start rsync command with exec
	errCreatePipe  = -10 // failed to create pipe for stdin/stdout/stderr
	errStartCmd    = -11 // failed to start the specified command
	errWaitProcess = -12 // failed to wait specified command process
)

var (
	// if get recoverable err of rsync, rsync wrapper will retry run rsync command
	recoverableErrList = [23]int{
		errSocketio,
		errFileio,
		errStreamio,
		errMessageio,
		errIPC,
		errCrashed,
		errTerminated,
		errSignal1,
		errSignal,
		errWaitChild,
		errMalloc,
		errPartial,
		errVanished,
		errDelLimit,
		errTimeout,
		errConTimeout,
		errCmdFailed,
		errCmdKilled,
		errCmdRun,
		errCmdNotfound,
		errCreatePipe,
		errStartCmd,
		errWaitProcess,
	}

	// if get unRecoverable err of rsync, rsync wrapper will exit direct,
	// also includes other errors that are unrecoverable and will cause the process to terminate.
	unRecoverableErrList = [5]int{
		errSyntax,
		errProtocol,
		errFileselect,
		errUnsupported,
		errStartclient,

		// other errors like:
		// errSIGKILL
		// errSIGBUS
		// errSIGSEGV
		// ......

	}

	exitCodeMap = map[int]int{
		errOK:          exit_code.Succeed,
		errSyntax:      exit_code.ErrInvalidArgument,
		errProtocol:    exit_code.ErrSystem,
		errFileselect:  exit_code.ErrNoSuchFileOrDir,
		errUnsupported: exit_code.ErrInvalidArgument,
		errStartclient: exit_code.ErrSystem,
		errSocketio:    exit_code.ErrIOError,
		errFileio:      exit_code.ErrIOError,
		errStreamio:    exit_code.ErrIOError,
		errMessageio:   exit_code.ErrIOError,
		errIPC:         exit_code.ErrSystem,
		errCrashed:     exit_code.ErrSystem,
		errTerminated:  exit_code.ErrSystem,
		errSignal1:     exit_code.ErrSystem,
		errSignal:      exit_code.ErrSystem,
		errWaitChild:   exit_code.ErrSystem,
		errMalloc:      exit_code.ErrSystem,
		errPartial:     exit_code.ErrPermissionDenied,
		errDelLimit:    exit_code.ErrSystem,
		errTimeout:     exit_code.ErrSystem,
		errConTimeout:  exit_code.ErrSystem,
		errCmdFailed:   exit_code.ErrSystem,
		errCmdKilled:   exit_code.ErrSystem,
		errCmdRun:      exit_code.ErrSystem,
		errCmdNotfound: exit_code.ErrSystem,
		errCreatePipe:  exit_code.ErrSystem,
		errStartCmd:    exit_code.ErrSystem,
		errWaitProcess: exit_code.ErrSystem,
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

// isErrUnRecoverable return true if err is unrecoverable.
func isErrUnRecoverable(errCode int) bool {
	for _, ec := range unRecoverableErrList {
		if ec == errCode {
			return true
		}
	}

	return false
}

// isWaitProcessErr return true if err return by Wait method of Process.
func isWaitProcessErr(err error) bool {
	if errors.Is(err, unix.EINVAL) {
		return true
	}

	_, ok := err.(*os.SyscallError)
	return ok
}

func exitCodeConvert(errCode int) int {
	exitCode, ok := exitCodeMap[errCode]
	if !ok {
		return exit_code.ErrSystem
	}

	return exitCode
}
