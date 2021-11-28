package exit_code

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
	ErrChecksumRefuse       = 201
	ErrUnknownFSType        = 202
	ErrCopylistPartial      = 252
	ErrCopyFileSucceed      = 254
	ErrSystem               = 255

	// std msg
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

	// custom msg
	ErrMsgSucceed         = "process succeed complete with exit code 0"
	ErrMsgSrcOrDest       = "src path or dest path is empty or not absolute path"
	ErrMsgFlagMissPartner = "missing required partner flag('progress' or 'stderr' miss partner 'report-addr')"
	ErrMsgReportAddr      = "report addr is unavailable"
	ErrMsgMaxLimitRetry   = "retry limit has been reached, but still get an error(can be recovered)"
	ErrMsgUnrecoverable   = "return a unrecoverable error"
)
