package user

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
	"transporter/pkg/filesystem"
)

const (
	stackBufCap       = 64 * 1024
	gdbBin            = "/usr/bin/gdb"
	gdbBatchMode      = "-batch"
	gdbExecutCommand  = "-ex"
	gdbDumpAllThread  = "thread apply all bt"
	gdbSpecifyProcess = "-p"
	slashChar         = '/'
	slashStr          = "/"
	userStackDir      = "/user/"
	permFileDefault   = 0644
)

var (
	ErrSaveDirUnavailableFormat = errors.New("unavailable format of save dir")
	ErrSaveDirNotDir            = errors.New("save dir that specified is not a dir")
)

func GoroutineStack() []byte {
	stackBuf := make([]byte, stackBufCap)
	n := runtime.Stack(stackBuf, true)
	stackBuf = stackBuf[:n]
	return stackBuf
}

func GoroutineStackFile(pid int32, saveDir string) error {
	if !filesystem.CheckDirPathFormat(saveDir) {
		return ErrSaveDirUnavailableFormat
	}

	saveDirInfo, err := os.Stat(saveDir)
	if err != nil {
		return err
	}

	if !saveDirInfo.IsDir() {
		return ErrSaveDirNotDir
	}

	if saveDir[len(saveDir)-1] != slashChar {
		saveDir += slashStr
	}
	processStackDir := saveDir + strconv.Itoa(int(pid)) + userStackDir
	err = filesystem.CheckOrCreateDir(processStackDir)
	if err != nil {
		return err
	}

	curCombinedStackFileName := time.Now().Format(time.RFC3339)
	curCombinedStackFilePath := processStackDir + slashStr + curCombinedStackFileName
	curCombinedStackFileInfo, err := os.OpenFile(
		curCombinedStackFilePath,
		unix.O_RDWR|unix.O_CREAT|unix.O_TRUNC|unix.O_APPEND,
		permFileDefault)
	if err != nil {
		log.Println(
			"[Stack-Error]Failed to open combined user stack file, err:",
			err.Error())
		return err
	}

	stackContent := GoroutineStack()
	_, err = curCombinedStackFileInfo.Write(stackContent)
	if err != nil {
		log.Println(
			"[Stack-Error]Failed to write combined user goroutine stack to combined file:",
			curCombinedStackFilePath,
			"err:", err.Error())
		_ = curCombinedStackFileInfo.Close()
		return err
	}
	log.Println("[Stack-Debug]Succeed to write combined user goroutine stack to combined file:", curCombinedStackFilePath)

	_ = curCombinedStackFileInfo.Close()
	return nil
}

func Stack(pid int32) ([]byte, error) {
	pidstr := strconv.Itoa(int(pid))
	c := exec.Command(gdbBin, gdbBatchMode, gdbExecutCommand, gdbDumpAllThread, gdbSpecifyProcess, pidstr)
	log.Println("[Stack-Debug]gdb cmd:", c.String())
	out, err := c.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return out, nil
}

func StackFile(pid int32, saveDir string) error {
	if !filesystem.CheckDirPathFormat(saveDir) {
		return ErrSaveDirUnavailableFormat
	}

	saveDirInfo, err := os.Stat(saveDir)
	if err != nil {
		return err
	}

	if !saveDirInfo.IsDir() {
		return ErrSaveDirNotDir
	}

	if saveDir[len(saveDir)-1] != slashChar {
		saveDir += slashStr
	}
	processStackDir := saveDir + strconv.Itoa(int(pid)) + userStackDir
	err = filesystem.CheckOrCreateDir(processStackDir)
	if err != nil {
		return err
	}

	curCombinedStackFileName := time.Now().Format(time.RFC3339)
	curCombinedStackFilePath := processStackDir + slashStr + curCombinedStackFileName
	curCombinedStackFileInfo, err := os.OpenFile(
		curCombinedStackFilePath,
		unix.O_RDWR|unix.O_CREAT|unix.O_TRUNC|unix.O_APPEND,
		permFileDefault)
	if err != nil {
		log.Println(
			"[Stack-Error]Failed to open combined user stack file, err:",
			err.Error())
		return err
	}

	stackContent, err := Stack(pid)
	if err != nil {
		log.Println(
			"[Stack-Error]Failed to get combined user stack:",
			curCombinedStackFilePath,
			"err:", err.Error())
		_ = curCombinedStackFileInfo.Close()
		return err
	}
	_, err = curCombinedStackFileInfo.Write(stackContent)
	if err != nil {
		log.Println(
			"[Stack-Error]Failed to write combined user stack to combined file:",
			curCombinedStackFilePath,
			"err:", err.Error())
		_ = curCombinedStackFileInfo.Close()
		return err
	}
	log.Println("[Stack-Debug]Succeed to write combined user stack to combined file:", curCombinedStackFilePath)

	_ = curCombinedStackFileInfo.Close()
	return nil
}
