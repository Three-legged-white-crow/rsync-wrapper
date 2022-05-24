package id

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"transporter/pkg/process"
)

const (
	pgrepBin               = "/usr/bin/pgrep"
	prgrepExitCodeMatched  = 0
	pgrepExitCodeNoMatched = 1
	pgrepExitCodeSyntaxErr = 2
	pgrepExitCodeFatalErr  = 3
	sepLF                  = "\n"
	sepSpace               = " "
)

func Current() int32 {
	return int32(os.Getpid())
}

func Children(pid int32) ([]process.Info, error) {
	c := exec.Command(pgrepBin, "-l", "-P", strconv.Itoa(int(pid)))
	var (
		out    bytes.Buffer
		outstr string
	)
	c.Stdout = &out
	err := c.Start()
	if err != nil {
		return nil, err
	}

	err = c.Wait()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			if exitErr.ExitCode() == pgrepExitCodeNoMatched {
				return nil, nil
			}
		}
		return nil, err
	}

	outstr = out.String()
	if len(outstr) == 0 {
		return nil, nil
	}

	lines := strings.Split(outstr, sepLF)
	res := make([]process.Info, 0, len(lines))
	var cpid int64

	for _, l := range lines {
		if len(l) == 0 {
			continue
		}

		ll := strings.Split(l, sepSpace)
		if len(ll) != 2 {
			continue
		}

		cpid, err = strconv.ParseInt(ll[0], 10, 32)
		if err != nil {
			continue
		}

		res = append(res, process.Info{
			Pid:  int32(cpid),
			Name: ll[1],
		})
	}

	return res, nil
}
