package rsync

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"fhyx/lib/log"
	"fhyx/lib/ssh"
)

type Options struct {
	// Archive, if true, enables archive mode.
	Archive bool

	// Delete, if true, deletes extraneous files from destination directories.
	Delete bool

	// Compress, if true, compresses file data during the transfer.
	Compress bool

	// Verbose, if true, increases rsync's verbosity.
	Verbose bool

	// Show progress during transfer
	Progress bool

	// Keep partially transferred files
	Partial bool

	// Stats bool
	Stats bool

	// Exclude contains files to be excluded from the transfer.
	Exclude []string

	// RemoteShell specifies the remote shell to use, e.g. ssh.
	RemoteShell string

	// RemoteHost specifies the remote host to copy files to/from.
	RemoteHost string

	// Additional options.
	Additional []string
}

type Rsync struct {
	Options *Options
	Info    Info
}

var (
	ErrSshpassNotExist  = errors.New("please install sshpass.")
	ErrRsyncNotExist    = errors.New("please install rsync.")
	ErrRsyncVersionOld  = errors.New("please upgrade rsync version.")
	ErrTmsRsyncNotExist = errors.New("tms has not install rsync.")
	ErrTmsNotFreeSpace  = errors.New("tms has not enough free space.")
)

func NewRsync(opt *Options, ssh SSH) (*Rsync, error) {
	if opt == nil {
		opt = DefaultOptions
	}
	opt.RemoteHost = ssh.Host
	opt.RemoteShell = ssh.Shell()
	return &Rsync{Options: opt}, nil
}

func CheckRsync(ssh SSH, targetPath string) error {
	if err := checkSrcRsync(ssh); err != nil {
		return err
	}
	if err := checkDstRsync(ssh, targetPath); err != nil {
		return err
	}
	return nil
}

func checkSrcRsync(s SSH) error {

	// check sshpass
	if s.Password != "" {
		cmd := exec.Command("sshpass", "-V")
		_, err := cmd.CombinedOutput()
		if err != nil {
			return ErrSshpassNotExist
		}
	}

	// check rsync
	cmd := exec.Command("rsync", "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ErrRsyncNotExist
	}

	// check rsync version
	versionMatcher := newMatcher(`rsync  version (\d).(\d).(\d)  protocol`)
	if versionMatcher.Match(string(out)) {
		v1 := versionMatcher.Extract(string(out), 1)
		v2 := versionMatcher.Extract(string(out), 2)
		if v1 < "3" || (v1 == "3" && v2 < "1") {
			return ErrRsyncVersionOld
		}
	}

	// check connect tms
	res, err := s.RunCommand(s.Command(nil))
	if err != nil {
		return errors.New(res)
	}

	return nil
}

func checkDstRsync(s SSH, targetPath string) error {

	client := ssh.NewSSHClient(s.Host, s.Port, s.User, s.Password, s.KeyFile)

	// check rsync
	_, err := client.RunCommand("rsync --version")
	if err != nil {
		return ErrTmsRsyncNotExist
	}

	// check disk usage
	if targetPath != "" {
		cmd := "df -k " + targetPath + " | awk '{printf(\"%d\",$4)}'"
		res, err := client.RunCommand(cmd)
		if err == nil {
			use, err := strconv.Atoi(res)
			if err == nil {
				if use < 10*1024 { // less 10M
					return ErrTmsNotFreeSpace
				}
			}
		}
	}

	return nil
}

func (r *Rsync) Copy(dst string, src ...string) error {

	r.Info = *NewInfo()

	// get stats
	r.Options.Stats = true
	r.Info.Cmd = r.command(dst, src...)
	err := r.Info.Run()
	if err != nil {
		return err
	}

	// copy file
	r.Options.Stats = false
	r.Info.Cmd = r.command(dst, src...)
	err = r.Info.Run()
	if err != nil {
		return err
	}

	// sync send and total num
	r.Info.State.Send = r.Info.State.Total
	r.Info.State.Progress = 100

	return nil
}

func (r *Rsync) command(dst string, src ...string) *exec.Cmd {
	args, err := r.Options.GetArgs(dst, src...)
	if err != nil {
		return nil
	}

	log.Infof("exec: %s\n", strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	return cmd
}

var DefaultOptions = &Options{
	Archive:  true,
	Compress: true,
	Partial:  true,
	Progress: true,
}

func (c *Options) GetArgs(dst string, src ...string) ([]string, error) {
	if len(src) == 0 {
		return nil, errors.New("no source given")
	}
	if dst == "" {
		return nil, errors.New("no destination given")
	}

	cmd := []string{"rsync"}

	if c.Archive {
		cmd = append(cmd, "--archive")
	}

	if c.Delete {
		cmd = append(cmd, "--delete")
	}

	if c.Compress {
		cmd = append(cmd, "--compress")
	}

	if c.Verbose {
		cmd = append(cmd, "--verbose")
	}

	if c.Progress {
		// cmd = append(cmd, "--progress")
		cmd = append(cmd, "--info=progress2")
		cmd = append(cmd, "--no-i-r")
	}

	if c.Stats {
		cmd = append(cmd, "--dry-run")
		cmd = append(cmd, "--stats")
	}

	if c.Partial {
		cmd = append(cmd, "--partial")
	}

	for _, x := range c.Exclude {
		cmd = append(cmd, "--exclude", x)
	}

	if c.RemoteShell != "" {
		if c.RemoteHost == "" {
			return nil, errors.New("no remote host given")
		}
		cmd = append(cmd, "--rsh", c.RemoteShell)
		dst = c.RemoteHost + ":" + dst
	}

	for _, o := range c.Additional {
		cmd = append(cmd, o)
	}

	cmd = append(cmd, src...)
	cmd = append(cmd, dst)
	return cmd, nil
}
