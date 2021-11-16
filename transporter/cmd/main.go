package main

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"log"
	"os"
	"strings"
	"time"

	"transporter/internal/filesystem"
	flag2 "transporter/internal/flag"
	"transporter/pkg/client"
	"transporter/pkg/exit_code"
	"transporter/pkg/rsync_wrapper"
)

const (
	slash            = "/"
	timeoutCreateDir = 120 // use second
)

func main() {

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)

	srcPath := flag.String(
		"src",
		"",
		"src path, use abs path")

	destPath := flag.String(
		"dest",
		"",
		"dest path, use abs path")

	isReportProgress := flag.Bool(
		"progress",
		false,
		"report progress of the transmission, must used with 'report-addr' flag")

	isReportStderr := flag.Bool(
		"stderr",
		false,
		"report std error content, must used with 'report-addr' flag")

	isSetDestDir := flag.Bool(
		"dest-dir",
		false,
		"treat dest as directoriy")

	addrReport := flag.String(
		"report-addr",
		"",
		"addr for report progress info or error message")

	intervalReport := flag.Int(
		"report-interval",
		0,
		"interval for report progress info, time unit is second, must positive integer")

	flag.Parse()

	isSrcAndDestAvailable := flag2.CheckSrcAndDest(*srcPath, *destPath)
	if !isSrcAndDestAvailable {
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("!![Info]Succeed to check src and dest")

	// Both 'progress' and 'report-addr' must be specified
	isBothProgressFlag := flag2.CheckProgressFlag(*isReportProgress, *addrReport)
	if !isBothProgressFlag {
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("!![Info]Succeed to check progress flag group")

	isBothStderrFlag := flag2.CheckStderrFlag(*isReportStderr, *addrReport)
	if !isBothStderrFlag {
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("!![Info]Succeed to check stderr flag group")

	isAddrReportAvailable := client.CheckAddr(*addrReport)
	if !isAddrReportAvailable {
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("!![Info]Succeed to check report address")

	if *isSetDestDir {

		ctx, cancelfunc := context.WithTimeout(context.Background(), timeoutCreateDir*time.Second)
		resChan := make(chan error, 1)
		go checkDestDir(*destPath, resChan)

		var (
			isCheck bool = false
			err     error
		)
		for {

			if isCheck {
				break
			}

			select {
			case <-ctx.Done():
				log.Println("!![Error]Timeout to check or create dest dir")
				// call cancelfunc is nouseful, because create dir is blocked, but also call
				cancelfunc()
				os.Exit(exit_code.ErrSystem)

			case err = <-resChan:
				if err != nil {
					log.Println("!![Error]Failed to check or create dest dir, err:", err.Error())
					if errors.Is(err, fs.ErrPermission) {
						os.Exit(exit_code.ErrPermissionDenied)
					}
					os.Exit(exit_code.ErrSystem)
				}

				log.Println("!![Info]Succeed to check or create dest dir")
				isCheck = true

			}
		}
	}

	var destPathCheck string
	destPathLastSlashIndex := strings.LastIndex(*destPath, slash)
	if destPathLastSlashIndex == 0 {
		destPathCheck = slash
	} else {
		destPathCheck = (*destPath)[:destPathLastSlashIndex]
	}
	errCheckMount := filesystem.IsMountPathList(*srcPath, destPathCheck)
	if errCheckMount != nil {
		log.Println(exit_code.ErrMsgCheckMount, ",err:", errCheckMount.Error())

		if errors.Is(errCheckMount, fs.ErrNotExist) {
			os.Exit(exit_code.ErrNoSuchFileOrDir)
		}

		if errors.Is(errCheckMount, fs.ErrPermission) {
			os.Exit(exit_code.ErrPermissionDenied)
		}

		os.Exit(exit_code.ErrSystem)
	}

	rc := client.NewReportClient()

	startTime := time.Now().String()
	log.Println("!![Info]Start at:", startTime)

	exitCode := rsync_wrapper.Run(*srcPath, *destPath, *addrReport, *isReportProgress, *isReportStderr, rc, *intervalReport)

	endTime := time.Now().String()
	log.Println("!![Info]End at:", endTime)

	// sleep a moment for wait all goroutine exit
	time.Sleep(5 * time.Second)
	os.Exit(exitCode)

}

func checkDestDir(destPath string, resChan chan<- error) {
	err := filesystem.CheckOrCreateDir(destPath)
	resChan <- err
	close(resChan)
}
