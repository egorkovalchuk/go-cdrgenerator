package pid

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"syscall"
)

func SetPID(logFileName string) error {
	filer, err := os.OpenFile(logFileName, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	filer.WriteString(fmt.Sprint(os.Getpid()))
	filer.Close()
	return nil
}

func RemovePID(logFileName string) error {
	return os.Remove(logFileName)
}

func StopProcess(pidFilePath string) error {
	if runtime.GOOS != "windows" {
		if _, err := os.Stat(pidFilePath); !os.IsNotExist(err) {
			raw, err := os.ReadFile(pidFilePath)
			if err != nil {
				return err
			}

			pid, err := strconv.Atoi(string(raw))
			if err != nil {
				return err
			}

			if proc, err := os.FindProcess(int(pid)); err == nil && !errors.Is(proc.Signal(syscall.Signal(15)), os.ErrProcessDone) {
				fmt.Fprintf(os.Stderr, "Process %d is stoping!\n", proc.Pid)

			} else if err = os.Remove(pidFilePath); err != nil {
				return err

			}
		}
	}
	return nil
}