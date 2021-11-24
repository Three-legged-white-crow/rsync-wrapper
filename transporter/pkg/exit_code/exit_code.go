package exit_code
/*
MD5VerifyRefuse       ErrCode = 1401
MD5VerifyRefuse:       "MD5 verify refuse"

在单文件子任务里，二次拷贝校验依然失败，退出码为： exit(201),  +1200=1401
在copy by list中，list file文件里，错误码直接是 1401
 */
const (
	Succeed                 = 0
	ErrNoSuchFileOrDir      = 2
	ErrIOError              = 5
	ErrPermissionDenied     = 13
	ErrDeviceIsBusy         = 16
	ErrFileIsExists         = 17
	ErrNotDirectory         = 20
	ErrInvalidArgument      = 22
	ErrNoSpaceLeftOnDevice  = 28
	ErrFileSystemIsReadOnly = 30
	ErrFileNameTooLong      = 36
	ErrChecksumRefuse       = 201
	ErrSystem               = 255

	ErrMsgSucceed         = "process succeed complete with exit code 0"
	ErrMsgSrcOrDest       = "src path or dest path is empty or not absolute path"
	ErrMsgFlagMissPartner = "missing required partner flag('progress' or 'stderr' miss partner 'report-addr')"
	ErrMsgReportAddr      = "report addr is unavailable"
	ErrMsgMaxLimitRetry   = "retry limit has been reached, but still get an error(can be recovered)"
	ErrMsgUnrecoverable   = "return a unrecoverable error"
	ErrMsgCheckMount      = "occur a error when check src or dest is mounted"
)
