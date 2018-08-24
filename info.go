package rsync

import (
	"io"
	"os/exec"
	"strconv"
	"strings"
)

type Info struct {
	Cmd        *exec.Cmd
	RsyncState RsyncState
	State      State
	Log        Log
}

type RsyncState struct {
	Total int64 `json:"total"`
	Trans int64 `json:"trans"`
}

type State struct {
	Send     int64   `json:"send"`
	Total    int64   `json:"total"`
	Speed    string  `json:"speed"`
	Progress float64 `json:"progress"`
}

type Log struct {
	Stderr string `json:"stderr"`
	Stdout string `json:"stdout"`
}

func NewInfo() *Info {
	return &Info{}
}

func (info *Info) Run() error {

	stdout, err := info.Cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()

	stderr, err := info.Cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer stderr.Close()

	go processStdout(info, stdout)
	go processStderr(info, stderr)

	return info.Cmd.Run()
}

var (
	// Total file size: 4,662,518,418 bytes
	// Total transferred file size: 1,555,052,808 bytes
	totalMatcher = newMatcher(`Total file size: (\d+(,\d+)*) bytes`)
	transMatcher = newMatcher(`Total transferred file size: (\d+(,\d+)*) bytes`)

	// 61,407,232   5%   58.56MB/s    0:00:16
	//120,324,096  11%   57.38MB/s    0:00:15
	progressMatcher = newMatcher(`(\d+(,\d+)*) +(\d+)%`)
	speedMatcher    = newMatcher(`(\d+\.\d+.{2}/s)`)
)

func processStdout(info *Info, stdout io.Reader) {

	buf := make([]byte, 2048, 2048)
	for {
		n, err := stdout.Read(buf[:])
		if n > 0 {

			logStr := string(buf[:n])
			// log.Println("\n*****\n", logStr, "\n/////\n")

			if totalMatcher.Match(logStr) {
				info.RsyncState.Total = convertBytesNum(totalMatcher.Extract(logStr, 1))
			}
			if transMatcher.Match(logStr) {
				info.RsyncState.Trans = convertBytesNum(transMatcher.Extract(logStr, 1))
			}

			if progressMatcher.Match(logStr) && info.RsyncState.Total > 0 {

				info.State.Total = info.RsyncState.Total

				partialSize := info.RsyncState.Total - info.RsyncState.Trans
				sendSize := convertBytesNum(progressMatcher.Extract(logStr, 1))
				info.State.Send = sendSize + partialSize

				info.State.Progress = float64(info.State.Send) / float64(info.State.Total) * 100
			}
			if speedMatcher.Match(logStr) {
				info.State.Speed = speedMatcher.Extract(logStr, 1)
			}

			info.Log.Stdout += logStr
		}
		if err != nil {
			break
		}
	}
}

func processStderr(info *Info, stderr io.Reader) {
	buf := make([]byte, 1024, 1024)
	for {
		n, err := stderr.Read(buf[:])
		if n > 0 {
			info.Log.Stderr += string(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

func convertBytesNum(str string) int64 {
	substr := strings.Replace(str, ",", "", -1)
	total, err := strconv.ParseInt(substr, 10, 64)
	if err != nil {
		return 0
	}
	return total
}
