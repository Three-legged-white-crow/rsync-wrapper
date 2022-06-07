package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"transporter/pkg/process/stack/user"
)

const emptyPid = -1

func main() {
	pid := flag.Int("p", emptyPid, "pid of process that need dump user stack")
	flag.Parse()

	if *pid == emptyPid {
		log.Printf("[Error]Please specify pid")
		return
	}

	res, err := user.Stack(int32(*pid))
	if err != nil {
		errExit, ok := err.(*exec.ExitError)
		if ok {
			log.Println("[Error]gdb exit but exit is not 0")
			log.Println("[Error]gdb cmd exit, exit code:", errExit.ExitCode())
			log.Println("[Error]gdb cmd exit, stderr:", string(errExit.Stderr))
			log.Println("[Error]gdb cmd exit, exit err str:", errExit.Error())
		} else {
			log.Println("[Error]failed to start gdb cmd, err:", err.Error())
		}
		log.Println("[Error]Failed to get user stack of process:", *pid, "and err:", err.Error())
		os.Exit(1)
	}

	log.Println("Succeed to get user stack of process:", *pid)
	fmt.Println(string(res))
}
