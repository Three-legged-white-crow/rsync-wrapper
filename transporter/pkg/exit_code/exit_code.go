package exit_code

const (
	Succeed = iota

	// ErrSrcOrDest means src path or dest path is empty or not absolute path.
	ErrSrcOrDest

	// ErrFlagMissPartner means missing required partner flag('-progress' and '-report-progress-addr' is a pair flags).
	ErrFlagMissPartner

	// ErrReportAddr means report addr is unavailable.
	ErrReportAddr

	// ErrCreateDestDir means occur a error when create dest directory.
	ErrCreateDestDir

	// ErrMaxLimitRetry means retry limit has been reached, but the Rsync still gives an error(can be recovered).
	ErrMaxLimitRetry

	// ErrUnrecoverable means rsync return a unrecoverable error.
	ErrUnrecoverable

	// ErrCheckMount means occur a error when get info of mounted filesystem.
	ErrCheckMount

	ErrMsgSucceed         = "process succeed complete with exit code 0"
	ErrMsgSrcOrDest       = "src path or dest path is empty or not absolute path"
	ErrMsgFlagMissPartner = "missing required partner flag('progress' or 'stderr' miss partner 'report-addr')"
	ErrMsgReportAddr      = "report addr is unavailable"
	ErrMsgCreateDestDir   = "occur a error when check or create dest directory"
	ErrMsgMaxLimitRetry   = "retry limit has been reached, but still get an error(can be recovered)"
	ErrMsgUnrecoverable   = "return a unrecoverable error"
	ErrMsgCheckMount      = "occur a error when check src or dest is mounted"
)
