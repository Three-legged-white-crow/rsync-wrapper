package checksum

const (
	Succeed                 = 0
	ErrNoSuchFileOrDir      = 2
	ErrIOError              = 5
	ErrPermissionDenied     = 13
	ErrDeviceIsBusy         = 16
	ErrInvalidArgument      = 22
	ErrNoSpaceLeftOnDevice  = 28
	ErrFileSystemIsReadOnly = 30
	ErrFileNameTooLong      = 36
	ErrSystem               = 255
)
