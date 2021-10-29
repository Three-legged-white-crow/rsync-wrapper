### rsync-wrapper

---

> rsync-wrapper 基于rsync封装，根据传入参数启动rsync程序执行拷贝，并对rsync的错误进行处理，对于可恢复错误会自动重试。



#### flags

- `src`

  - string

  - 源路径，要求是绝对路径

  - 默认为空

    

- `dest`

  - string

  - 目的路径，要求是绝对路径

  - 默认为空

    

- `dest-dir`

  - bool

  - 将`dest`路径当作目录对待

  - 默认为false

    

- `progress`

  - bool

  - 是否获取拷贝进度并上报，指定此参数时必须同时指定`report-addr`参数

  - 默认为false

    

- `stderr`

  - bool
  -  是否获取rsync标准错误输出并上报，指定此参数时必须同时指定`report-addr`参数
  - 默认为false
  
  
  
- `report-addr`

  - string
  - 将会把拷贝进度信息和程序错误信息上报到该地址
  - 默认为空



#### examples

- 只上报拷贝进度信息
  - `/usr/local/transporter -dest-dir -progress -report-addr http://api/report -src /src -dest /dest`
- 只上报程序错误信息
  - `/usr/local/transporter -dest-dir -stderr -report-addr http://api/report -src /src -dest /dest`
- 不上报任何信息
  - `/usr/local/transporter -dest-dir -src /src -dest /dest`

#### exit code

```go
package exit_code

const (
	Succeed = iota       // 0

	// ErrSrcOrDest means src path or dest path is empty or not absolute path.
	ErrSrcOrDest         // 1

	// ErrFlagMissPartner means missing required partner flag('-progress' and '-report-progress-addr' is a pair flags).
	ErrFlagMissPartner   // 2

	// ErrReportAddr means report addr is unavailable.
	ErrReportAddr        // 3

	// ErrCreateDestDir means occur a error when create dest directory.
	ErrCreateDestDir     // 4

	// ErrMaxLimitRetry means retry limit has been reached, but the Rsync still gives an error(can be recovered).
	ErrMaxLimitRetry     // 5

	// ErrUnrecoverable means rsync return a unrecoverable error.
	ErrUnrecoverable     // 6

	ErrMsgSucceed         = "process succeed complete with exit code 0"
	ErrMsgSrcOrDest       = "src path or dest path is empty or not absolute path"
	ErrMsgFlagMissPartner = "missing required partner flag('progress' or 'stderr' miss partner 'report-addr')"
	ErrMsgReportAddr      = "report addr is unavailable"
	ErrMsgCreateDestDir   = "occur a error when check or create dest directory"
	ErrMsgMaxLimitRetry   = "retry limit has been reached, but still get an error(can be recovered)"
	ErrMsgUnrecoverable   = "return a unrecoverable error"
)
```



#### 上报数据结构

```go
 type reqResult struct {
    Count   int64  `json:"count"`   // progress number
    Message string `json:"message"` // rsync stderr content
    ErrCode int64  `json:"errcode"` // exit code
    Reason  string `json:"reason"`  // reason of exit error
 }

```



#### rsync 错误分类

```go
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
        errSignal      = 20  // received SIGINT, SIGTERM, or SIGHUP
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
	recoverableErrList = []int{
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

	// if get unRecoverable err of rsync, rsync wrapper will exit direct
	unRecoverableErrList = []int{
		errSyntax,
		errProtocol,
		errFileselect,
		errUnsupported,
		errStartclient,
	}

)

```



#### Container and Signal

```
                  system pid namespace
systemd (pid=1)                         
   |
   | —— docker-containerd-shim
               |
               |       container pid namespace
               |    -----------------------------
               | —— transporter (pid=1, ppid=0) -
                    -    |                      - 
                    -    | —— rsync             - 
                    -----------------------------
```

- docker通过调用带有`CLONE_NEWPID`标志的[`clone()`](http://man7.org/linux/man-pages/man2/clone.2.html)在container中创建了一个新的 PID namespace。在container内部1号进程transporter会忽略 SIGTERM和SIGKILL，而transporter进程在fork and exec启动rsync进程后，会wait rsync进程，一旦rsync进程被SIGTERM或SIGKILL信号杀死，transporter进程的wait调用就会返回。
- 当OOM或人为在Host向transporter进程发送SIGTERM或SIGKILL信号杀死该进程时，container的1号进程退出，container也会退出，而container中的其他进程也会收到SIGKILL信号被杀死。
- 所以transporter进程和rsync进程总是一同存活或者被杀死，不会出现rsync进程转变为孤儿进程的可能。
- 参考：
  - [Namespaces in operation, part 3: PID namespaces](https://lwn.net/Articles/531419/)
  - [谁是Docker容器的init(1)进程](https://shareinto.github.io/2019/01/30/docker-init(1)/)
  - [What happens to other processes when a Docker container's PID1 exits?](https://stackoverflow.com/questions/39739658/what-happens-to-other-processes-when-a-docker-containers-pid1-exits)
