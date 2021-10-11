package flag

import (
	"log"

	"transporter/pkg/exit_code"
)

// CheckSrcAndDest check src path and dest path is available.
func CheckSrcAndDest(src, dest string) bool {
	if len(src) == 0 || len(dest) == 0 {
		log.Println(exit_code.ErrMsgSrcOrDest)
		log.Println("src path:", src, "dest path:", dest)
		return false
	}

	if src[0] != '/' || dest[0] != '/' {
		log.Println(exit_code.ErrMsgSrcOrDest)
		log.Println("src path:", src, "dest path:", dest)
		return false
	}

	return true
}

// CheckProgressFlag check flag pair of progress is both specified.
func CheckProgressFlag(isReportProgress bool, addrReportProgress string) bool {
	if (isReportProgress && len(addrReportProgress) > 0) || !isReportProgress {
		return true
	}

	log.Println(exit_code.ErrMsgFlagMissPartner)
	log.Println("report progress:", isReportProgress, "report addr:", addrReportProgress)
	return false
}

// CheckStderrFlag check flag pair of stderr is both specified.
func CheckStderrFlag(isReportStderr bool, addReportStderr string) bool {
	if (isReportStderr && len(addReportStderr) > 0) || !isReportStderr {
		return true
	}

	log.Println(exit_code.ErrMsgFlagMissPartner)
	log.Println("report stderr:", isReportStderr, "report addr:", addReportStderr)
	return false
}
