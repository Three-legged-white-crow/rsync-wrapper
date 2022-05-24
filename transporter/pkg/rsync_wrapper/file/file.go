package file

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
	"transporter/pkg/exit_code"
	"transporter/pkg/filesystem"
	"transporter/pkg/process"
	"transporter/pkg/process/id"
	"transporter/pkg/process/stack"
	"transporter/pkg/rsync_wrapper"
)

const (
	rsyncBinPath       = "/usr/local/bin/rsync"
	rsyncOptionBasic   = "-rlptgoHA"
	rsyncOptionPartial = "--partial"
	rsyncOptionSparse  = "--sparse"
	retryMaxLimit      = 3
	stackBasePath      = "/var/log/rsync-wrapper-stack/"
	intervalDumpStack  = 3600 // second
	timeoutWaitDump    = 30   // second
	dumpStateRunning   = 1
	dumpStateSleeping  = 2
	envSRMTaskID       = "SRM_TASK_ID"
	commandRsyncSubStr = "rsync"
	slashChar          = '/'
	slashStr           = "/"
	permFileDefault    = 0644
)

var (
	dumpState uint32 = dumpStateSleeping
)

type ReqContent struct {
	SrcPath        string
	DestPath       string
	IsHandleSparse bool
	RetryLimit     int
	RecordStack    bool
}

func isDumpRunning() bool {
	curDumpState := atomic.LoadUint32(&dumpState)
	if curDumpState == dumpStateRunning {
		return true
	}

	return false
}

func waitDumpComplete() {
	if !isDumpRunning() {
		return
	}

	time.Sleep(timeoutWaitDump * time.Second)
	return
}

func dumpStack() {
	defer func() {
		panicErr := recover()
		if panicErr != nil {
			log.Println(
				"[CopyFile-Warning]Get panic error when dump stack and err:",
				panicErr)
		}
	}()

	taskID := os.Getenv(envSRMTaskID)
	if len(taskID) == 0 {
		log.Println("[CopyFile-Warning]Task id from env is empty, abort dump stack!")
		return
	}

	var (
		err error
	)

	saveDir := stackBasePath + taskID
	err = filesystem.CheckOrCreateDir(saveDir)
	if err != nil {
		log.Println(
			"[CopyFile-Warning]Failed to create save dir:", saveDir,
			"and err:", err.Error(),
			", abort dump stack!")
		return
	}

	wrapperPid := id.Current()
	log.Println(
		"[CopyFile-Info]Start goroutine for dump stack, pid:", wrapperPid,
		"save dir:", saveDir)

	// process tree:
	// - rsync-wrapper           --> wrapper
	//   - rsync sender          --> rsync
	//     - rsync receiver      --> rsync
	//       - rsync generator   --> rsync

	var (
		childrens          []process.Info
		child              process.Info
		senderPid          int32
		receiverPid        int32
		generatorPid       int32
		curDumpProcessTree reqRecordProcessTree
	)

	for {
		atomic.StoreUint32(&dumpState, dumpStateSleeping)
		time.Sleep(intervalDumpStack * time.Second)
		atomic.StoreUint32(&dumpState, dumpStateRunning)

		// combined rsync-wrapper process stack;
		// current process is rsync-wrapper process;
		err = stack.CombinedStack(wrapperPid, saveDir)
		if err != nil {
			log.Println(
				"[CopyFile-Warning]Failed to get combined stack info of wrapper process, task:", taskID,
				", wrapper pid:", wrapperPid,
				", saveDir:", saveDir,
				" and err:", err.Error())
			continue
		}
		log.Println(
			"[CopyFile-Info]Succeed to dump combined stack of wrapper process, task:", taskID,
			", wrapper pid:", wrapperPid,
			"and saveDir:", saveDir)

		// combined rsync sender process stack;
		// sender process is children of rsync-wrapper process;
		childrens, err = id.Children(wrapperPid)
		if err != nil {
			log.Println(
				"[CopyFile-Warning]Failed to get children of wrapper process:", wrapperPid,
				"and err:", err.Error())
			continue
		}
		// the first child process that command contain 'rsync' is sender process
		for _, child = range childrens {
			if strings.Contains(child.Name, commandRsyncSubStr) {
				senderPid = child.Pid
				break
			}
		}
		if senderPid == 0 {
			log.Println(
				"[CopyFile-Warning]Can not find rsync sender process, wrapper pid:", wrapperPid,
				"and childrens:", childrens)
			continue
		}

		err = stack.CombinedStack(senderPid, saveDir)
		if err != nil {
			log.Println(
				"[CopyFile-Warning]Failed to get combined stack info of sender process, task:", taskID,
				", sender pid:", senderPid,
				", saveDir:", saveDir,
				" and err:", err.Error())
			continue
		}
		log.Println("[CopyFile-Info]Succeed to get combined stack of sender process, task:", taskID,
			", sender pid:", senderPid,
			"and saveDir:", saveDir)

		// combined rsync receiver process stack;
		// receiver process is children of sender process;
		childrens, err = id.Children(senderPid)
		if err != nil {
			log.Println(
				"[CopyFile-Warning]Failed to get children of sender process:", senderPid,
				"and err:", err.Error())
			continue
		}

		for _, child = range childrens {
			if strings.Contains(child.Name, commandRsyncSubStr) {
				receiverPid = child.Pid
				break
			}
		}
		if receiverPid == 0 {
			log.Println(
				"[CopyFile-Warning]Can not find rsync receiver process, sender pid:", senderPid,
				"and childrens:", childrens)
			continue
		}
		err = stack.CombinedStack(receiverPid, saveDir)
		if err != nil {
			log.Println(
				"[CopyFile-Warning]Failed to get combined stack info of receiver process, task:", taskID,
				", receiver pid:", receiverPid,
				", saveDir:", saveDir,
				" and err:", err.Error())
			continue
		}
		log.Println("[CopyFile-Info]Succeed to get combined stack of receiver process, task:", taskID,
			", receiver pid:", receiverPid,
			"and saveDir:", saveDir)

		// combined rsync generator process stack;
		// generator process is children of receiver process;
		childrens, err = id.Children(receiverPid)
		if err != nil {
			log.Println(
				"[CopyFile-Warning]Failed to get children of receiver process:", receiverPid,
				"and err:", err.Error())
			continue
		}
		for _, child = range childrens {
			if strings.Contains(child.Name, commandRsyncSubStr) {
				generatorPid = child.Pid
				break
			}
		}
		if generatorPid == 0 {
			log.Println(
				"[CopyFile-Warning]Can not find rsync generator process, receiver pid:", receiverPid,
				"and childrens:", childrens)
			continue
		}
		err = stack.CombinedStack(generatorPid, saveDir)
		if err != nil {
			log.Println(
				"[CopyFile-Warning]Failed to get combined stack info of generator process, task:", taskID,
				", generator pid:", generatorPid,
				", saveDir:", saveDir,
				" and err:", err.Error())
			continue
		}
		log.Println("[CopyFile-Info]Succeed to get combined stack of generator process, task:", taskID,
			", generator pid:", generatorPid,
			"and saveDir:", saveDir)

		// record process tree at current dump loop
		curDumpProcessTree = reqRecordProcessTree{
			saveDir:      saveDir,
			wrapperPid:   wrapperPid,
			senderPid:    senderPid,
			receiverPid:  receiverPid,
			generatorPid: generatorPid,
		}
		err = recordProcessTree(curDumpProcessTree)
		if err != nil {
			log.Println("[CopyFile-Wraning]Failed to record process tree, and err:", err.Error())
			continue
		}
		log.Println("[CopyFile-Info]Succeed to record process tree")
	}
}

type reqRecordProcessTree struct {
	saveDir      string
	wrapperPid   int32
	senderPid    int32
	receiverPid  int32
	generatorPid int32
}

func recordProcessTree(req reqRecordProcessTree) error {
	if req.wrapperPid == 0 {
		return errors.New("wrapper process pid is empty")
	}

	if req.senderPid == 0 {
		return errors.New("sender process pid is empty")
	}

	if req.receiverPid == 0 {
		return errors.New("receiver process pid is empty")
	}

	if req.generatorPid == 0 {
		return errors.New("generator process pid is empty")
	}

	if !filesystem.CheckDirPathFormat(req.saveDir) {
		return errors.New("unavailable format of save dir for record process tree")
	}

	saveDirInfo, err := os.Stat(req.saveDir)
	if err != nil {
		return err
	}

	if !saveDirInfo.IsDir() {
		return errors.New("save dir is not dir")
	}

	if req.saveDir[len(req.saveDir)-1] != slashChar {
		req.saveDir = req.saveDir + slashStr
	}

	recordFileName := time.Now().Format(time.RFC3339)
	recordFilePath := req.saveDir + recordFileName
	recordFileInfo, err := os.OpenFile(
		recordFilePath,
		unix.O_RDWR|unix.O_CREAT|unix.O_TRUNC|unix.O_APPEND,
		permFileDefault)
	if err != nil {
		return err
	}
	recordBuilder := strings.Builder{}
	recordBuilder.WriteString(recordFileName)
	recordBuilder.WriteString("\n")
	recordBuilder.WriteString("wrapper pid:")
	recordBuilder.WriteString(strconv.Itoa(int(req.wrapperPid)))
	recordBuilder.WriteString("\n")
	recordBuilder.WriteString("  sender pid:")
	recordBuilder.WriteString(strconv.Itoa(int(req.senderPid)))
	recordBuilder.WriteString("\n")
	recordBuilder.WriteString("    receiver pid:")
	recordBuilder.WriteString(strconv.Itoa(int(req.receiverPid)))
	recordBuilder.WriteString("\n")
	recordBuilder.WriteString("      generator pid:")
	recordBuilder.WriteString(strconv.Itoa(int(req.generatorPid)))
	recordBuilder.WriteString("\n")
	_, err = recordFileInfo.WriteString(recordBuilder.String())
	if err != nil {
		_ = recordFileInfo.Close()
		return err
	}

	_ = recordFileInfo.Close()
	return nil
}

func CopyFile(req ReqContent) int {

	var (
		finalExitCode     int
		ok                bool
		currentRetryLimit int
		currentRetryNum   int
		stdoutStderr      []byte
		err               error
	)

	currentRetryLimit = req.RetryLimit
	if currentRetryLimit < 0 {
		currentRetryLimit = retryMaxLimit
	}

	cmdContent := []string{rsyncOptionBasic, rsyncOptionPartial}
	if req.IsHandleSparse {
		cmdContent = append(cmdContent, rsyncOptionSparse)
	}

	cmdContent = append(cmdContent, req.SrcPath)
	cmdContent = append(cmdContent, req.DestPath)

	for {
		if currentRetryNum > currentRetryLimit {
			log.Println("[CopyFile-Error]Retry limit reached, exit with ErrRetryLImit(208)")
			return exit_code.ErrRetryLimit
		}

		c := exec.Command(rsyncBinPath, cmdContent...)
		log.Println("[CopyFile-Info]Run command:", c.String(), "retry number:", currentRetryNum)

		go dumpStack()
		stdoutStderr, err = c.CombinedOutput()
		if err == nil {
			waitDumpComplete()
			return exit_code.Succeed
		}

		if errors.Is(err, unix.EINVAL) {
			waitDumpComplete()
			return exit_code.ErrInvalidArgument
		}

		var processExitErr *exec.ExitError
		processExitErr, ok = err.(*exec.ExitError)
		if !ok {
			log.Println(
				"[CopyFile-Error]Get err but not ExitError when copy src:", req.SrcPath,
				"to dest:", req.DestPath,
				"and err:", err.Error())
			waitDumpComplete()
			return exit_code.ErrSystem
		}
		log.Println(
			"[CopyFile-Error]Get subprocess exit error:", processExitErr.Error(),
			"exit code:", processExitErr.ExitCode(),
			"subprocess pid", processExitErr.Pid())

		finalExitCode, ok = rsync_wrapper.ExitCodeConvertWithStderr(string(stdoutStderr))
		if ok {
			log.Println("[CopyFile-Error]Matched stand file system exit code and exit:", finalExitCode)
			waitDumpComplete()
			return finalExitCode
		}

		processExitCode := processExitErr.ExitCode()
		if rsync_wrapper.IsErrUnRecoverable(processExitCode) {
			log.Println(
				"[CopyFile-Error]Get unrecoverable err when copy src:", req.SrcPath,
				"to dest:", req.DestPath,
				"and err:", processExitErr.Error())
			finalExitCode = rsync_wrapper.ExitCodeConvert(processExitCode)
			waitDumpComplete()
			return finalExitCode
		}

		currentRetryNum += 1
		log.Println("[CopyFile-Warning]Get exit error but it is a recoverable error, will retry exec command")
		continue
	}

}
