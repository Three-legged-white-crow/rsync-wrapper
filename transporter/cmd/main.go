package main

import (
	"flag"
	"log"
	"os"
	"time"

	flag2 "transporter/internal/flag"
	"transporter/pkg/client"
	"transporter/pkg/exit_code"
	"transporter/pkg/rsync_wrapper"
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
		os.Exit(exit_code.ErrSrcOrDest)
	}

	// Both 'progress' and 'report-addr' must be specified
	isBothProgressFlag := flag2.CheckProgressFlag(*isReportProgress, *addrReport)
	if !isBothProgressFlag {
		os.Exit(exit_code.ErrFlagMissPartner)
	}

	isBothStderrFlag := flag2.CheckStderrFlag(*isReportStderr, *addrReport)
	if !isBothStderrFlag {
		os.Exit(exit_code.ErrFlagMissPartner)
	}

	isAddrReportAvailable := client.CheckAddr(*addrReport)
	if !isAddrReportAvailable {
		os.Exit(exit_code.ErrReportAddr)
	}

	if *isSetDestDir {
		err := rsync_wrapper.CheckOrCreateDir(*destPath)
		if err != nil {
			log.Println(exit_code.ErrMsgCreateDestDir)
			log.Println("dest path:", *destPath, "err:", err.Error())
			os.Exit(exit_code.ErrCreateDestDir)
		}
	}

	rc := client.NewReportClient()

	startTime := time.Now().String()
	log.Println("Start at:", startTime)

	exitCode := rsync_wrapper.Run(*srcPath, *destPath, *addrReport, *isReportProgress, *isReportStderr, rc, *intervalReport)

	endTime := time.Now().String()
	log.Println("End at:", endTime)

	// sleep a moment for wait all goroutine exit
	time.Sleep(5 * time.Second)
	os.Exit(exitCode)

}
