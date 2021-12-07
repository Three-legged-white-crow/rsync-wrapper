package main

import (
	"bufio"
	"errors"
	"flag"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	"transporter/pkg/checksum"
	"transporter/pkg/exit_code"
	"transporter/pkg/filesystem"
	"transporter/pkg/rsync_wrapper/file"
)

const (
	emptyValue              = "empty"
	slash                   = '/'
	slashStr                = "/"
	waitNFSCliUpdate        = 5
	waitNFSCcliLimit        = 5
	delimLF                 = '\n'
	delimLFStr              = "\n"
	delimCRLFStr            = "\r\n"
	delimCRStr              = "\r"
	seq                     = ","
	minLimitRecord          = 2
	maxLimitRecord          = 3
	srcRelativeRecordIndex  = 0
	destRelativeRecordIndex = 1
	errcodeRecordIndex      = 2
	recordDirtyChar         = '"'
	recordDirtyStr          = `"`
	permFileDefault         = 0775
	errCodeAdditional       = 1200
)

func main() {

	srcMountPath := flag.String(
		"src-mount",
		emptyValue,
		"mount point path of src")

	destMountPath := flag.String(
		"dest-mount",
		emptyValue,
		"mount point path of dest")

	inputRecordFile := flag.String(
		"record-file-in",
		emptyValue,
		"input record file path relative dest mount point")

	outputRecordFile := flag.String(
		"record-file-out",
		emptyValue,
		"output record file path relative dest mount point")

	isIgnoreSrcNotExist := flag.Bool(
		"ignore-src-not-exist",
		false,
		"if src file is not exist will skip error, otherwise record error to file that 'record-file-out' specified")

	isIgnoreSrcIsDir := flag.Bool(
		"ignore-src-dir",
		false,
		"if src file is dir will skip error, otherwise record error to file that 'record-file-out' specified")

	isIgnoreDestIsExistDir := flag.Bool(
		"ignore-dest-dir",
		false,
		"if dest is exist dir will skip error, otherwise record error to file that 'record-file-out' specified")

	isOverwriteDestFile := flag.Bool(
		"overwrite-dest-file",
		false,
		"overwirte dest exist file, if dest is dir, do nothing")

	isGenerateChecksumFile := flag.Bool(
		"generate-checksum-file",
		false,
		"generate checksum file to same dir as dest, if dest is dir, do nothing")

	fileSuffixForChecksum := flag.String(
		"checksum-suffix",
		emptyValue,
		"suffix of file that need checksum")

	trackFileRelativePath := flag.String(
		"track-file",
		emptyValue,
		"track file relative to the dest mount point")

	isRemoveInRecordFile := flag.Bool(
		"remove-input-file",
		false,
		"remove input record file")

	isDebug := flag.Bool(
		"debug",
		false,
		"enable debug mode")

	flag.Parse()

	// set output of standard logger to stderr
	log.SetOutput(os.Stderr)

	log.Println("[copylist-Info]New transporter request, srcMountPath:", *srcMountPath,
		"destMountPath:", *destMountPath,
		"inputRecordFile:", *inputRecordFile,
		"outputRecordFile:", *outputRecordFile,
		"isIgnoreSrcNotExist:", *isIgnoreSrcNotExist,
		"isIgnoreSrcIsDir:", *isIgnoreSrcIsDir,
		"isIgnoreDestIsExistDir:", *isIgnoreDestIsExistDir,
		"isOverwriteDestFile:", *isOverwriteDestFile,
		"isGenerateChecksumFile:", *isGenerateChecksumFile,
		"trackFileRelativePath:", *trackFileRelativePath,
		"fileSuffixForChecksum:", *fileSuffixForChecksum,
		"isRemoveInRecordFile:", *isRemoveInRecordFile,
		"isDebug:", *isDebug,
	)

	log.Println("[copylist-Info]Start check")
	log.Println("[copylist-Info]Start check format")

	var (
		isPathAvailable        bool
		inRecordFilePath       string
		outRecordFilePath      string
		err                    error
		exitCode               int
		isChecksumSuffixEmpty  bool
		checksumFileSuffixList []string
		isCreateTrackFile      bool
		trackFilePath          string
	)
	isPathAvailable = filesystem.CheckDirPathFormat(*srcMountPath)
	if !isPathAvailable {
		log.Println("[copylist-Error]Unavailable format of src mount point:", *srcMountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckDirPathFormat(*destMountPath)
	if !isPathAvailable {
		log.Println("[copylist-Error]Unavailable format of dest mount point:", *destMountPath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	if *fileSuffixForChecksum == emptyValue {
		isChecksumSuffixEmpty = true
		log.Println("[copylist-Info]Not specify checksum suffix, not checksum")
	} else {
		checksumFileSuffixList = strings.Split(*fileSuffixForChecksum, slashStr)
	}

	if *trackFileRelativePath != emptyValue {
		isCreateTrackFile = true
	}

	// not need check err, because format of mount point has already been checked above
	inRecordFilePath, _ = filesystem.AbsolutePath(*destMountPath, *inputRecordFile)
	outRecordFilePath, _ = filesystem.AbsolutePath(*destMountPath, *outputRecordFile)

	if isCreateTrackFile {
		trackFilePath, _ = filesystem.AbsolutePath(*destMountPath, *trackFileRelativePath)
		log.Println("[copylist-Info]Need create track file:", trackFilePath)

		isPathAvailable = filesystem.CheckFilePathFormat(trackFilePath)
		if !isPathAvailable {
			log.Println("[copylist-Error]Unavailable format of track file:", trackFilePath)
			os.Exit(exit_code.ErrInvalidArgument)
		}
		log.Println("[copylist-Info]Check track file format...OK")
	}

	isPathAvailable = filesystem.CheckFilePathFormat(inRecordFilePath)
	if !isPathAvailable {
		log.Println("[copylist-Error]Unavailable format of input record file:", inRecordFilePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}

	isPathAvailable = filesystem.CheckFilePathFormat(outRecordFilePath)
	if !isPathAvailable {
		log.Println("[copylist-Error]Unavailable format of output record file:", outRecordFilePath)
		os.Exit(exit_code.ErrInvalidArgument)
	}
	log.Println("[copylist-Info]Check format...OK")

	if !(*isDebug) {
		log.Println("[copylist-Info]Start check mount filesystem")
		err = filesystem.IsMountPath(*srcMountPath)
		if err != nil {
			log.Println(
				"[copylist-Error]Failed to check src mount filesystem:", *srcMountPath,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		err = filesystem.IsMountPath(*destMountPath)
		if err != nil {
			log.Println(
				"[copylist-Error]Failed to check dest mount filesystem:", *destMountPath,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
		log.Println("[copylist-Info]Check mount filesystem...OK")
	}

	/*
		if input file not exist -> ENOENT
		if input file exist:
			- parse input file and load src/dest relative path -> build abs path -> check src -> next
			- src is not exist
				- if isIgnoreSrcNotExist is false -> record ENOENT as reason to output file -> next
				- if isIgnoreSrcNotExist is ture -> next record
			- src is exist
				- if src is dir:
					- if isIgnoreSrcIsDir is flase -> record EISDIR as reason to output file -> next
					- if isIgnoreSrcIsDir is true -> next record

		src is file -> check dest
			- if dest is exist:
				- if dest is dir:
					- if isIgnoreDestIsExistDir is false -> record EISDIR as reason to output file -> next
					- if isIgnoreDestIsExistDir is true ->  next record
				- if dest is file
					- if isOverwriteDestFile is false -> record EEXIST as reason to output file -> next
					- if isOverwriteDestFile is true -> trancunt dest file

		- start copy file -> get exit code of rsync:
			- if not succeed -> record to output file -> next
		- checksum src and dest
			- if not equal -> remove dest file and dest checksum file -> retry copy -> retry checksum
			- if equal
				- if isGenerateChecksumFile is true -> generate checksum file -> next
				- if isGenerateChecksumFile is false -> next
			- if failed to checksum src and dest again -> record ErrChecksumRefuse as reason to output file -> next

		- read EOF of input file -> remove input file
		- if record something to output file -> ErrCopylistPartial(252)
		- if not record something to output file -> Succeed(0)

	*/

	log.Println("[copylist-Info]Start check input record file is exist")

	var (
		inputRecordFileInfo os.FileInfo
		retryNum            int
	)
	for {
		if retryNum >= waitNFSCcliLimit {
			log.Println("[copylist-Error]Input record file:", inRecordFilePath,
				"is not exit, retry num:", retryNum)
			os.Exit(exit_code.ErrNoSuchFileOrDir)
		}

		time.Sleep(waitNFSCliUpdate * time.Second)
		inputRecordFileInfo, err = os.Stat(inRecordFilePath)
		if err == nil {
			if inputRecordFileInfo.IsDir() {
				log.Println("[copylist-Error]Input record file is exist, but is dir:", inRecordFilePath)
				os.Exit(exit_code.ErrIsDirectory)
			}

			break
		}

		if !errors.Is(err, fs.ErrNotExist) {
			log.Println("[copylist-Error]Failed to stat input record file:", inRecordFilePath,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}

		retryNum += 1
	}
	log.Println("[copylist-Info]Check input record file is exist...Exist")

	log.Println("[copylist-Info]Start check record format of input file")
	inputF, err := os.Open(inRecordFilePath)
	if err != nil {
		log.Println("[copylist-Error]Failed to open(1) input record file:", inRecordFilePath,
			"and err:", err.Error())
		exitCode = exit_code.ExitCodeConvertWithErr(err)
		os.Exit(exitCode)
	}

	// check record format of input file
	var (
		line              string
		isRecordAvailable bool
	)
	inputReader := bufio.NewReader(inputF)
	for {
		line, err = inputReader.ReadString(delimLF)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Println("[copylist-Error]Get err when read input record file:", inRecordFilePath,
					"and err:", err.Error())

				_ = inputF.Close()
				exitCode = exit_code.ExitCodeConvertWithErr(err)
				os.Exit(exitCode)
			}

			log.Println("[copylist-Error]Read EOF of input record file:", inRecordFilePath)
			break
		}

		isRecordAvailable = checkRecord(line)
		if !isRecordAvailable {
			_ = inputF.Close()
			log.Println("[copylist-Error]Unavailable record: >>", line, "<<")
			os.Exit(exit_code.ErrInvalidListFile)
		}
	}
	_ = inputF.Close()
	log.Println("[copylist-Info]Check record format of input file...OK")

	log.Println("[copylist-Info]Start parse input record file, reopen input file")
	// reopen input file to parse record
	inputF, err = os.Open(inRecordFilePath)
	if err != nil {
		log.Println("[copylist-Error]Failed to open(2) input record file:", inRecordFilePath,
			"and err:", err.Error())
		exitCode = exit_code.ExitCodeConvertWithErr(err)
		os.Exit(exitCode)
	}
	inputReader.Reset(inputF)

	// check or create output record file, if parent dir is not exist, create it
	err = filesystem.CheckOrCreateFile(outRecordFilePath, true)
	if err != nil {
		log.Println("[copylist-Error]Failed to check or create output record file:", outRecordFilePath,
			"and err:", err.Error())
		_ = inputF.Close()
		exitCode = exit_code.ExitCodeConvertWithErr(err)
		os.Exit(exitCode)
	}

	var outputF *os.File
	outputF, err = os.OpenFile(outRecordFilePath, unix.O_RDWR|unix.O_CREAT|unix.O_TRUNC|unix.O_APPEND, permFileDefault)
	if err != nil {
		log.Println("[copylist-Error]Failed to open output record file:", outRecordFilePath,
			"and err:", err.Error())
		_ = inputF.Close()
		exitCode = exit_code.ExitCodeConvertWithErr(err)
		os.Exit(exitCode)
	}

	var (
		isRecordErr    bool = false
		outputWriter   *bufio.Writer
		srcPath        string
		destPath       string
		srcPathInfo    os.FileInfo
		destPathInfo   os.FileInfo
		lastSlashIndex int
		destFileName   string
		destParentDir  string
		exitCodeStr    string
		recordBuilder  strings.Builder
		recordErrStr   string
		recordContent  recordInfo
		firstExitCode  int = exit_code.Empty
		numRecord      int
		numErrRecord   int
		numIgSrcDir    int
		numIgSrcNOENT  int
		numIgDestDir   int
		numOverWrite   int
	)

	outputWriter = bufio.NewWriter(outputF)
	for {
		line, err = inputReader.ReadString(delimLF)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Println("[copylist-Error]Get err when read input record file:", inRecordFilePath,
					"and err:", err.Error())

				_ = inputF.Close()
				_ = outputF.Close()

				exitCode = exit_code.ExitCodeConvertWithErr(err)
				os.Exit(exitCode)
			}

			log.Println("[copylist-Info]Read end of input record file:", inRecordFilePath,
				"and get err:", err.Error())
			break
		}
		numRecord += 1

		recordContent, isRecordAvailable = cleanRecord(line)
		if !isRecordAvailable {
			numErrRecord += 1
			isRecordErr = true
			recordBuilder.Reset()
			recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
			recordBuilder.WriteString(seq)
			recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
			recordBuilder.WriteString(seq)
			exitCodeStr = strconv.Itoa(exit_code.ErrInvalidListFile + errCodeAdditional)
			recordBuilder.WriteString(exitCodeStr)
			recordBuilder.WriteString("\n")
			recordErrStr = recordBuilder.String()
			_, _ = outputWriter.WriteString(recordErrStr)
			if firstExitCode == exit_code.Empty {
				firstExitCode = exit_code.ErrInvalidListFile
			}
			continue
		}
		srcPath, _ = filesystem.AbsolutePath(*srcMountPath, recordContent.srcRelativeCleanPath)
		destPath, _ = filesystem.AbsolutePath(*destMountPath, recordContent.destRelativeCleanPath)

		if *isDebug {
			log.Println("[copylist-debug]srcRelativePath:", recordContent.srcRelativeCleanPath,
				"destRelativePath:", recordContent.destRelativeCleanPath,
				"srcPath:", srcPath,
				"destPath:", destPath)
		}

		isPathAvailable = filesystem.CheckFilePathFormat(srcPath)
		if !isPathAvailable {
			numErrRecord += 1
			isRecordErr = true
			recordBuilder.Reset()
			recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
			recordBuilder.WriteString(seq)
			recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
			recordBuilder.WriteString(seq)
			exitCodeStr = strconv.Itoa(exit_code.ErrInvalidArgument + errCodeAdditional)
			recordBuilder.WriteString(exitCodeStr)
			recordBuilder.WriteString("\n")
			recordErrStr = recordBuilder.String()
			_, _ = outputWriter.WriteString(recordErrStr)
			if firstExitCode == exit_code.Empty {
				firstExitCode = exit_code.ErrInvalidArgument
			}
			continue
		}
		isPathAvailable = filesystem.CheckFilePathFormat(destPath)
		if !isPathAvailable {
			numErrRecord += 1
			isRecordErr = true
			recordBuilder.Reset()
			recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
			recordBuilder.WriteString(seq)
			recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
			recordBuilder.WriteString(seq)
			exitCodeStr = strconv.Itoa(exit_code.ErrInvalidArgument + errCodeAdditional)
			recordBuilder.WriteString(exitCodeStr)
			recordBuilder.WriteString("\n")
			recordErrStr = recordBuilder.String()
			_, _ = outputWriter.WriteString(recordErrStr)
			if firstExitCode == exit_code.Empty {
				firstExitCode = exit_code.ErrInvalidArgument
			}
			continue
		}

		srcPathInfo, err = os.Stat(srcPath)
		if err != nil {
			numErrRecord += 1

			if !errors.Is(err, fs.ErrNotExist) {
				isRecordErr = true
				recordBuilder.Reset()
				recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
				recordBuilder.WriteString(seq)
				recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
				recordBuilder.WriteString(seq)
				exitCode = exit_code.ExitCodeConvertWithErr(err)
				if exitCode == exit_code.ErrSystem {
					exitCodeStr = strconv.Itoa(exit_code.SystemError)
				} else {
					exitCodeStr = strconv.Itoa(exitCode + errCodeAdditional)
				}
				recordBuilder.WriteString(exitCodeStr)
				recordBuilder.WriteString("\n")
				recordErrStr = recordBuilder.String()
				_, _ = outputWriter.WriteString(recordErrStr)
				if firstExitCode == exit_code.Empty {
					firstExitCode = exitCode
				}
				continue
			}

			if !(*isIgnoreSrcNotExist) {
				isRecordErr = true
				recordBuilder.Reset()
				recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
				recordBuilder.WriteString(seq)
				recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
				recordBuilder.WriteString(seq)
				exitCodeStr = strconv.Itoa(exit_code.ErrNoSuchFileOrDir + errCodeAdditional)
				recordBuilder.WriteString(exitCodeStr)
				recordBuilder.WriteString("\n")
				recordErrStr = recordBuilder.String()
				_, _ = outputWriter.WriteString(recordErrStr)
				if firstExitCode == exit_code.Empty {
					firstExitCode = exit_code.ErrNoSuchFileOrDir
				}
			} else {
				numIgSrcNOENT += 1
			}
			continue
		}

		if srcPathInfo.IsDir() {
			numErrRecord += 1

			if !(*isIgnoreSrcIsDir) {
				isRecordErr = true
				recordBuilder.Reset()
				recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
				recordBuilder.WriteString(seq)
				recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
				recordBuilder.WriteString(seq)
				exitCodeStr = strconv.Itoa(exit_code.ErrIsDirectory + errCodeAdditional)
				recordBuilder.WriteString(exitCodeStr)
				recordBuilder.WriteString("\n")
				recordErrStr = recordBuilder.String()
				_, _ = outputWriter.WriteString(recordErrStr)
				if firstExitCode == exit_code.Empty {
					firstExitCode = exit_code.ErrIsDirectory
				}
			} else {
				numIgSrcDir += 1
			}
			continue
		}

		// src and dest are same file
		if srcPath == destPath {
			numErrRecord += 1
			isRecordErr = true
			recordBuilder.Reset()
			recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
			recordBuilder.WriteString(seq)
			recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
			recordBuilder.WriteString(seq)
			exitCodeStr = strconv.Itoa(exit_code.ErrSrcAndDstAreSameFile + errCodeAdditional)
			recordBuilder.WriteString(exitCodeStr)
			recordBuilder.WriteString("\n")
			recordErrStr = recordBuilder.String()
			_, _ = outputWriter.WriteString(recordErrStr)
			if firstExitCode == exit_code.Empty {
				firstExitCode = exit_code.ErrSrcAndDstAreSameFile
			}
			continue
		}

		// src is exist file, let's check dest
		destPathInfo, err = os.Stat(destPath)
		if err == nil {
			if destPathInfo.IsDir() {
				numErrRecord += 1
				if !(*isIgnoreDestIsExistDir) {
					isRecordErr = true
					recordBuilder.Reset()
					recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
					recordBuilder.WriteString(seq)
					recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
					recordBuilder.WriteString(seq)
					exitCodeStr = strconv.Itoa(exit_code.ErrIsDirectory + errCodeAdditional)
					recordBuilder.WriteString(exitCodeStr)
					recordBuilder.WriteString("\n")
					recordErrStr = recordBuilder.String()
					_, _ = outputWriter.WriteString(recordErrStr)
					if firstExitCode == exit_code.Empty {
						firstExitCode = exit_code.ErrIsDirectory
					}
				} else {
					numIgDestDir += 1
				}
				continue
			}

			// dest is exist file
			if !(*isOverwriteDestFile) {
				numErrRecord += 1
				isRecordErr = true
				recordBuilder.Reset()
				recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
				recordBuilder.WriteString(seq)
				recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
				recordBuilder.WriteString(seq)
				exitCodeStr = strconv.Itoa(exit_code.ErrFileIsExists + errCodeAdditional)
				recordBuilder.WriteString(exitCodeStr)
				recordBuilder.WriteString("\n")
				recordErrStr = recordBuilder.String()
				_, _ = outputWriter.WriteString(recordErrStr)
				if firstExitCode == exit_code.Empty {
					firstExitCode = exit_code.ErrFileIsExists
				}
				continue
			} else {
				numOverWrite += 1
			}

			// trunc dest file
			_, err = os.OpenFile(destPath, unix.O_RDWR|unix.O_TRUNC, permFileDefault)
			if err != nil {
				numErrRecord += 1
				isRecordErr = true
				recordBuilder.Reset()
				recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
				recordBuilder.WriteString(seq)
				recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
				recordBuilder.WriteString(seq)
				exitCode = exit_code.ExitCodeConvertWithErr(err)
				if exitCode == exit_code.ErrSystem {
					exitCodeStr = strconv.Itoa(exit_code.SystemError)
				} else {
					exitCodeStr = strconv.Itoa(exitCode + errCodeAdditional)
				}
				recordBuilder.WriteString(exitCodeStr)
				recordBuilder.WriteString("\n")
				recordErrStr = recordBuilder.String()
				_, _ = outputWriter.WriteString(recordErrStr)
				if firstExitCode == exit_code.Empty {
					firstExitCode = exitCode
				}
				continue
			}

			// trunc dest file succeed
		} else {
			if !errors.Is(err, fs.ErrNotExist) {
				numErrRecord += 1
				isRecordErr = true
				recordBuilder.Reset()
				recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
				recordBuilder.WriteString(seq)
				recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
				recordBuilder.WriteString(seq)
				exitCode = exit_code.ExitCodeConvertWithErr(err)
				if exitCode == exit_code.ErrSystem {
					exitCodeStr = strconv.Itoa(exit_code.SystemError)
				} else {
					exitCodeStr = strconv.Itoa(exitCode + errCodeAdditional)
				}
				recordBuilder.WriteString(exitCodeStr)
				recordBuilder.WriteString("\n")
				recordErrStr = recordBuilder.String()
				_, _ = outputWriter.WriteString(recordErrStr)
				if firstExitCode == exit_code.Empty {
					firstExitCode = exitCode
				}
				continue
			}

			// not exist
		}

		// check or create dest parent dir
		lastSlashIndex = strings.LastIndex(destPath, slashStr)
		destFileName = destPath[lastSlashIndex+1:]
		destParentDir = strings.TrimSuffix(destPath, destFileName)

		if *isDebug {
			log.Println("[copylist-debug]Last slash index:", lastSlashIndex)
			log.Println("[copylist-debug]Dest file name:", destFileName)
			log.Println("[copylist-debug]Dest parent dir:", destParentDir)
		}

		err = filesystem.CheckOrCreateDir(destParentDir)
		if err != nil {
			numErrRecord += 1
			isRecordErr = true
			recordBuilder.Reset()
			recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
			recordBuilder.WriteString(seq)
			recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
			recordBuilder.WriteString(seq)
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			if exitCode == exit_code.ErrSystem {
				exitCodeStr = strconv.Itoa(exit_code.SystemError)
			} else {
				exitCodeStr = strconv.Itoa(exitCode + errCodeAdditional)
			}
			recordBuilder.WriteString(exitCodeStr)
			recordBuilder.WriteString("\n")
			recordErrStr = recordBuilder.String()
			_, _ = outputWriter.WriteString(recordErrStr)
			if firstExitCode == exit_code.Empty {
				firstExitCode = exitCode
			}
			continue
		}

		exitCode = file.CopyFile(srcPath, destPath)
		if exitCode != exit_code.Succeed {
			// try remove dest file to clean dest to clean dest
			_ = os.Remove(destPath)
			numErrRecord += 1
			isRecordErr = true
			recordBuilder.Reset()
			recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
			recordBuilder.WriteString(seq)
			recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
			recordBuilder.WriteString(seq)
			if exitCode == exit_code.ErrSystem {
				exitCodeStr = strconv.Itoa(exit_code.SystemError)
			} else {
				exitCodeStr = strconv.Itoa(exitCode + errCodeAdditional)
			}
			recordBuilder.WriteString(exitCodeStr)
			recordBuilder.WriteString("\n")
			recordErrStr = recordBuilder.String()
			_, _ = outputWriter.WriteString(recordErrStr)
			if firstExitCode == exit_code.Empty {
				firstExitCode = exitCode
			}
			continue
		}

		if !isChecksumSuffixEmpty && isNeedChecksum(srcPathInfo.Name(), checksumFileSuffixList) {

			err = checksum.MD5Checksum(srcPath, destPath, *isGenerateChecksumFile)
			if err != nil {
				// internal retry again
				err = os.Remove(destPath)
				if err != nil {
					numErrRecord += 1
					isRecordErr = true
					recordBuilder.Reset()
					recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
					recordBuilder.WriteString(seq)
					recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
					recordBuilder.WriteString(seq)
					exitCode = exit_code.ExitCodeConvertWithErr(err)
					if exitCode == exit_code.ErrSystem {
						exitCodeStr = strconv.Itoa(exit_code.SystemError)
					} else {
						exitCodeStr = strconv.Itoa(exitCode + errCodeAdditional)
					}
					recordBuilder.WriteString(exitCodeStr)
					recordBuilder.WriteString("\n")
					recordErrStr = recordBuilder.String()
					_, _ = outputWriter.WriteString(recordErrStr)
					if firstExitCode == exit_code.Empty {
						firstExitCode = exitCode
					}
					continue
				}

				exitCode = file.CopyFile(srcPath, destPath)
				if exitCode != exit_code.Succeed {
					// try remove dest file and checksum file to clean dest
					_ = os.Remove(destPath)
					_ = os.Remove(destPath + checksum.MD5Suffix)
					numErrRecord += 1
					isRecordErr = true
					recordBuilder.Reset()
					recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
					recordBuilder.WriteString(seq)
					recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
					recordBuilder.WriteString(seq)
					if exitCode == exit_code.ErrSystem {
						exitCodeStr = strconv.Itoa(exit_code.SystemError)
					} else {
						exitCodeStr = strconv.Itoa(exitCode + errCodeAdditional)
					}
					recordBuilder.WriteString(exitCodeStr)
					recordBuilder.WriteString("\n")
					recordErrStr = recordBuilder.String()
					_, _ = outputWriter.WriteString(recordErrStr)
					if firstExitCode == exit_code.Empty {
						firstExitCode = exitCode
					}
					continue
				}

				err = checksum.MD5Checksum(srcPath, destPath, *isGenerateChecksumFile)
				if err != nil {
					// try remove dest file and checksum file to clean dest
					_ = os.Remove(destPath)
					_ = os.Remove(destPath + checksum.MD5Suffix)
					numErrRecord += 1
					isRecordErr = true
					recordBuilder.Reset()
					recordBuilder.WriteString(recordContent.srcRelativeDirtyPath)
					recordBuilder.WriteString(seq)
					recordBuilder.WriteString(recordContent.destRelativeDirtyPath)
					recordBuilder.WriteString(seq)
					exitCodeStr = strconv.Itoa(exit_code.ErrChecksumRefuse + errCodeAdditional)
					recordBuilder.WriteString(exitCodeStr)
					recordBuilder.WriteString("\n")
					recordErrStr = recordBuilder.String()
					_, _ = outputWriter.WriteString(recordErrStr)
					if firstExitCode == exit_code.Empty {
						firstExitCode = exit_code.ErrChecksumRefuse
					}
					continue
				}

			}
		}
	}

	err = outputWriter.Flush()
	if err != nil {
		log.Println("[copylist-Warning]Failed to flush any buffered data to the underlying io.Writer, and err:", err.Error())
	} else {
		log.Println("[copylist-Info]Succeed to flush any buffered data to the underlying io.Writer")
	}

	err = outputF.Sync()
	if err != nil {
		log.Println("[copylist-Warning]Failed to sync content of file to storage, and err:", err.Error())
	} else {
		log.Println("[copylist-Info]Succeed to sync content of file to storage")
	}

	err = inputF.Close()
	if err != nil {
		log.Println("[copylist-Warning]Failed to close input record file, and err:", err.Error())
	} else {
		log.Println("[copylist-Warning]Succeed to close input record file")
	}

	err = outputF.Close()
	if err != nil {
		log.Println("[copylist-Warning]Failed to close output record file, and err:", err.Error())
	} else {
		log.Println("[copylist-Warning]Succeed to close output record file")
	}

	if *isRemoveInRecordFile {
		err = os.Remove(inRecordFilePath)
		if err != nil {
			log.Println("[copylist-Warning]Failed to remove input record file:", inRecordFilePath,
				"and err:", err.Error())
		} else {
			log.Println("[copylist-Info]Succeed to remove input record file:", inRecordFilePath)
		}
	}

	if isCreateTrackFile {
		err = filesystem.CheckOrCreateFile(trackFilePath, false)
		if err != nil {
			log.Println("[copylist-Error]Failed to check or create track file:", trackFilePath,
				"and err:", err.Error())
			exitCode = exit_code.ExitCodeConvertWithErr(err)
			os.Exit(exitCode)
		}
		log.Println("[copylist-Info]Succeed to create track file:", trackFilePath)
	}

	if isRecordErr {
		log.Println("[copylist-Warning]Record some error to output record file:", outRecordFilePath)
	}

	log.Println("[copylist-Info]Number total ->",
		"total record:", numRecord,
		"err record:", numErrRecord,
		"ignore src is dir:", numIgSrcDir,
		"ignore src not exist:", numIgSrcNOENT,
		"ignore dest is dir:", numIgDestDir,
		"overWrite:", numOverWrite)
	if numRecord == numErrRecord {
		if firstExitCode != exit_code.Empty {
			log.Println("[copylist-Error]All records get err, exit with first err:", firstExitCode)
			os.Exit(firstExitCode)
		}

		log.Println("[copylist-Info]All records get err, but all ignore, exit with 0")
		os.Exit(exit_code.Succeed)
	}

	if isRecordErr {
		log.Println("[copylist-Error]Some records get err, exit with",
			exit_code.ErrCopylistPartial, "(ErrCopylistPartial)")
		os.Exit(exit_code.ErrCopylistPartial)
	}

	log.Println("[copylist-Info]No error record, exit with:", exit_code.Succeed)
	os.Exit(exit_code.Succeed)

}

/*
	stand format:
	format 1: "srctest/dir1/file1","desttest/dir2/file2"
	format 2: "srctest/dir1/file1","desttest/dir2/file2",1202

	use LF (\n) as a line break
*/
func checkRecord(record string) bool {
	if strings.Contains(record, delimCRLFStr) {
		log.Println("[copylist-Error]Use unsupport delim: CRLF")
		return false
	}

	if strings.Contains(record, delimCRStr) {
		log.Println("[copylist-Error]Use unsupport delim: CR")
		return false
	}

	if !strings.HasSuffix(record, delimLFStr) {
		return false
	}

	recordList := strings.Split(record, seq)
	if len(recordList) < minLimitRecord || len(recordList) > maxLimitRecord {
		log.Println("[copylist-Error]Record column < 2 or > 3")
		return false
	}

	var ok bool

	ok = checkRecordPath(recordList[srcRelativeRecordIndex])
	if !ok {
		return false
	}

	ok = checkRecordPath(recordList[destRelativeRecordIndex])
	if !ok {
		return false
	}

	return true
}

func checkRecordPath(path string) bool {
	numBoundaryMarker := strings.Count(path, recordDirtyStr)
	if numBoundaryMarker != 2 {
		log.Println("[copylist-Error]Num of quotation marks != 2")
		return false
	}

	return true
}

type recordInfo struct {
	srcRelativeDirtyPath  string
	srcRelativeCleanPath  string
	destRelativeDirtyPath string
	destRelativeCleanPath string
}

func cleanRecord(record string) (recordInfo, bool) {

	record = strings.TrimSuffix(record, delimLFStr)

	recordList := strings.Split(record, seq)
	if len(recordList) < minLimitRecord || len(recordList) > maxLimitRecord {
		return recordInfo{}, false
	}

	srcDirty, srcClean, ok := cleanRecordPath(recordList[srcRelativeRecordIndex])
	if !ok {
		return recordInfo{}, false
	}

	destDirty, destClean, ok := cleanRecordPath(recordList[destRelativeRecordIndex])
	if !ok {
		return recordInfo{}, false
	}

	return recordInfo{
		srcRelativeDirtyPath:  srcDirty,
		srcRelativeCleanPath:  srcClean,
		destRelativeDirtyPath: destDirty,
		destRelativeCleanPath: destClean,
	}, true
}

func cleanRecordPath(path string) (dirtyPath string, cleanPath string, isAvailable bool) {
	ok := checkRecordPath(path)
	if !ok {
		return "", "", false
	}

	firstIndex := strings.Index(path, recordDirtyStr)
	cleanPath = path[firstIndex+1:]

	lastIndex := strings.LastIndex(cleanPath, recordDirtyStr)
	cleanPath = cleanPath[:lastIndex]

	dirtyPath = recordDirtyStr + cleanPath + recordDirtyStr

	return dirtyPath, cleanPath, true
}

func isNeedChecksum(fileName string, fileSuffixList []string) bool {
	if fileSuffixList == nil {
		return false
	}

	var isMatch bool
	for _, suffix := range fileSuffixList {
		isMatch, _ = filepath.Match(suffix, fileName)
		if isMatch {
			return true
		}
	}
	return false
}
