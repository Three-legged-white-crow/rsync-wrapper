package stack

import (
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	"transporter/pkg/filesystem"
)

const (
	procBase             = "/proc/"
	procProcessTasks     = "/task/"
	procProcessStackFile = "/stack"
	defaultLimtReadDir   = 100
	slashChar            = '/'
	slashStr             = "/"
	permFileDefault      = 0644
)

var (
	ErrSaveDirUnavailableFormat = errors.New("unavailable format of save dir")
	ErrSaveDirNotDir            = errors.New("save dir that specified is not a dir")
)

func GetStackPathList(pid int32) ([]string, error) {
	tasksDir := procBase + strconv.Itoa(int(pid)) + procProcessTasks
	tf, err := os.Open(tasksDir)
	if err != nil {
		log.Println("[Stack-Error]Failed to open task of process:", pid, "err:", err.Error())
		return nil, err
	}

	var (
		taskNameList      []string
		taskStackPathList []string
		taskStackPath     string
	)

	for {
		taskNameList, err = tf.Readdirnames(defaultLimtReadDir)
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Println("[Stack-Info]Get EOF when read task name list of process:", pid)
				break
			}

			log.Println("[Stack-Error]Failed to read task name list of process:", pid, "err:", err.Error())
			return nil, err
		}

		for _, tn := range taskNameList {
			taskStackPath = tasksDir + tn + procProcessStackFile
			taskStackPathList = append(taskStackPathList, taskStackPath)
		}
	}

	return taskStackPathList, nil
}

func CombinedStack(pid int32, saveDir string) error {
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

	taskStackPathList, err := GetStackPathList(pid)
	if err != nil {
		return err
	}
	log.Println("[Stack-Debug]Get task stack path list:", taskStackPathList, "of pid:", pid)

	if saveDir[len(saveDir)-1] != slashChar {
		saveDir += slashStr
	}
	processStackDir := saveDir + strconv.Itoa(int(pid))
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
			"[Stack-Error]Failed to open combined stack file, err:",
			err.Error())
		return err
	}

	var (
		stackBuilder strings.Builder
		stackContent []byte
	)

	for _, taskStackPath := range taskStackPathList {
		stackContent, err = os.ReadFile(taskStackPath)
		if err != nil {
			log.Println("[Stack-Warning]Failed to read stack of thread:", taskStackPath)
			continue
		}

		stackBuilder.WriteString("theard: ")
		stackBuilder.WriteString(taskStackPath)
		stackBuilder.WriteString("\n")
		stackBuilder.Write(stackContent)
		stackBuilder.WriteString("\n")
	}

	_, err = curCombinedStackFileInfo.WriteString(stackBuilder.String())
	if err != nil {
		log.Println(
			"[Stack-Error]Failed to write combined stack to combined file:",
			curCombinedStackFilePath,
			"err:", err.Error())
		_ = curCombinedStackFileInfo.Close()
		return err
	}
	log.Println("[Stack-Debug]Succeed to write combined stack to combined file:", curCombinedStackFilePath)

	_ = curCombinedStackFileInfo.Close()
	return nil
}
