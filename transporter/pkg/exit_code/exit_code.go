//go:build amd64 && linux
// +build amd64,linux

package exit_code

import (
	"errors"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
	"transporter/pkg/checksum"
	"transporter/pkg/filesystem"
)

// linux std error code, only include filesystem related
const (
	Succeed                 = 0
	ErrNoSuchFileOrDir      = 2
	ErrIOError              = 5
	ErrPermissionDenied     = 13
	ErrDeviceIsBusy         = 16
	ErrFileIsExists         = 17
	ErrNotDirectory         = 20
	ErrIsDirectory          = 21
	ErrInvalidArgument      = 22
	ErrNoSpaceLeftOnDevice  = 28
	ErrFileSystemIsReadOnly = 30
	ErrFileNameTooLong      = 36
	ErrFileStale            = 116
	ErrDiskQuota            = 122
)

// desc of linux std error code, only include filesystem related
const (
	ErrMsgNOENT             = "No such file or directory"
	ErrMsgIOError           = "Input/output error"
	ErrMsgPermissionDenided = "Permission denied"
	ErrMsgDeviceBusy        = "Device or resource busy"
	ErrMsgFileIsExists      = "File exists"
	ErrMsgNotDirectory      = "Not a directory"
	ErrMsgIsDirectory       = "Is a directory"
	ErrMsgInval             = "Invalid argument"
	ErrMsgNoSpace           = "No space left on device"
	ErrMsgFSReadOnly        = "Read-only filesystem"
	ErrMsgDiskQuota         = "Disk quota exceeded"
	ErrMsgFileStale         = "Stale file handle"
)

// custom error code
const (
	ErrChecksumRefuse        = 201
	ErrUnknownFSType         = 202
	ErrSrcAndDstAreSameFile  = 203
	ErrDirectoryNestedItself = 204
	ErrInvalidListFile       = 205
	ErrRetryLimit            = 208
	ErrCopylistPartial       = 252
	ErrCopyFileSucceed       = 254
	ErrSystem                = 255
	Empty                    = -999
)

// custom msg
const (
	ErrMsgSucceed         = "process succeed complete with exit code 0"
	ErrMsgSrcOrDest       = "src path or dest path is empty or not absolute path"
	ErrMsgFlagMissPartner = "missing required partner flag('progress' or 'stderr' miss partner 'report-addr')"
	ErrMsgReportAddr      = "report addr is unavailable"
	ErrMsgMaxLimitRetry   = "retry limit has been reached, but still get an error(can be recovered)"
	ErrMsgUnrecoverable   = "return a unrecoverable error"
)

// error code of api
const (
	SystemError           = 1001
	WrongRequestBody      = 1002
	WrongRequestPath      = 1003
	JsonMarshalError      = 1004
	JsonUnMarshalError    = 1005
	WrongResponseBody     = 1006
	FailedToRouteRequest  = 1007
	OperationNotPermitted = 1201
	NoSuchFileOrDir       = 1202
	NoSuchProcess         = 1203
	InterruptedSystemCall = 1204
	IOError               = 1205
	NoSuchDeviceOrAddress = 1206
	ArgListTooLong        = 1207
	ExecFormatError       = 1208
	BadFileNumber         = 1209
	NoChildProcesses      = 1210
	TryAgain              = 1211
	OutOfMemory           = 1212
	PermissionDenied      = 1213
	BadAddress            = 1214
	BlockDeviceRequired   = 1215
	DeviceIsBusy          = 1216
	FileIsExists          = 1217
	CrossDeviceLink       = 1218
	NoSuchDevice          = 1219
	NotDirectory          = 1220
	IsDirectory           = 1221
	InvalidArgument       = 1222
	FileTableOverflow     = 1223
	TooManyOpenFiles      = 1224
	NotTypewriter         = 1225
	TextFileBusy          = 1226
	FileTooLarge          = 1227
	NoSpaceLeftOnDevice   = 1228
	IllegalSeek           = 1229
	FileSystemIsReadOnly  = 1230
	TooManyLinks          = 1231
	BrokenPipe            = 1232
	FileNameTooLong       = 1236
	DirNotEmpty           = 1239
	TooManySymbolic       = 1240
	TransportNotConnected = 1307
	StaleFileHandle       = 1316
	QuotaExceeded         = 1322
	MD5VerifyRefuse       = 1401
	UnknownFsType         = 1402
	SrcAndDstAreSameFile  = 1403
	DirectoryNestedItself = 1404
	InvalidListFile       = 1405
	RetryLimit            = 1408
)

func ExitCodeConvertWithErr(err error) int {
	if err == nil {
		return Succeed
	}

	if errors.Is(err, fs.ErrNotExist) {
		return ErrNoSuchFileOrDir
	}

	if errors.Is(err, fs.ErrPermission) {
		return ErrPermissionDenied
	}

	if errors.Is(err, fs.ErrExist) {
		return ErrFileIsExists
	}

	if errors.Is(err, fs.ErrInvalid) {
		return ErrInvalidArgument
	}

	if errors.Is(err, filesystem.ErrUnavailableFileSystem) {
		return ErrUnknownFSType
	}

	if errors.Is(err, checksum.ErrNotEqual) {
		return ErrChecksumRefuse
	}

	var realErr error
	switch realErr1 := err.(type) {
	case *os.PathError:
		realErr = realErr1.Err
	case *os.LinkError:
		realErr = realErr1.Err
	case *os.SyscallError:
		realErr = realErr1.Err
	default:
		realErr = err
	}

	realErrNum, ok := realErr.(unix.Errno)
	if !ok {
		return ErrSystem
	}

	return int(realErrNum)

}
