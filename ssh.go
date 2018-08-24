package rsync

import (
	"errors"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

type SSH struct {
	ConfigFile string
	Host       string
	User       string
	Port       int
	Password   string
	KeyFile    string
	Options    []string
}

// The host string has the format [user@]hostname[:port]
func NewSSH(host string) (*SSH, error) {
	var user string
	a := strings.Split(host, "@")
	if len(a) > 1 {
		user = a[0]
		host = a[1]
	}

	var port int
	a = strings.Split(host, ":")
	if len(a) > 1 {
		host = a[0]
		var err error
		if port, err = strconv.Atoi(a[1]); err != nil {
			return nil, errors.New("invalid SSH port")
		}
	}
	return &SSH{Host: host, User: user, Port: port}, nil
}

func (s *SSH) Command(args []string) []string {

	cmd := []string{"ssh"}
	if s.Password != "" {
		cmd = []string{"sshpass"}
		cmd = append(cmd, "-p", s.Password)
		cmd = append(cmd, "ssh")
	}

	cmd = append(cmd, "-T")

	if s.ConfigFile != "" {
		cmd = append(cmd, "-F", s.ConfigFile)
	}

	if s.User != "" {
		cmd = append(cmd, "-l", s.User)
	}

	if s.Port != 0 {
		cmd = append(cmd, "-p", strconv.Itoa(s.Port))
	}

	if s.KeyFile != "" {
		cmd = append(cmd, "-i", s.KeyFile)
	}

	for _, o := range s.Options {
		cmd = append(cmd, "-o", o)
	}

	if s.Host != "" {
		cmd = append(cmd, s.Host)
	}

	if len(args) > 0 {
		cmd = append(cmd, args...)
	}

	return cmd
}

func (s *SSH) RunCommand(args []string) (string, error) {
	if len(args) == 0 {
		e := "no command given"
		return e, errors.New(e)
	}
	if s.Host == "" {
		e := "no host given"
		return e, errors.New(e)
	}

	log.Printf("exec: %s\n", strings.Join(args, " "))
	res, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	return string(res), err
}

func (s *SSH) Shell() string {
	cmd := s.Command([]string{})
	var quoted []string
	for _, i := range cmd[:len(cmd)-1] {
		quoted = append(quoted, "'"+i+"'")
	}
	return strings.Join(quoted, " ")
}
