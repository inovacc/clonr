package process

import (
	"strings"

	"github.com/google/gops/goprocess"
)

type Process struct {
	procList []Process

	PID  int
	Exec string
	Path string
	Name string
}

func NewProcess() *Process {
	return &Process{}
}

func (p *Process) ListProcesses() error {
	processes := goprocess.FindAll()
	for _, proc := range processes {
		p.procList = append(p.procList, Process{
			PID:  proc.PID,
			Exec: proc.Exec,
			Path: proc.Path,
		})
	}

	return nil
}

func (p *Process) IsProcessRunning(pid int) bool {
	for _, proc := range p.procList {
		if proc.PID == pid {
			return true
		}
	}

	return false
}

func (p *Process) ProcessExists(pid int, name string) bool {
	for _, proc := range p.procList {
		if proc.PID == pid {
			return strings.Contains(strings.ToLower(proc.Name), name) || strings.Contains(strings.ToLower(proc.Path), name)
		}
	}

	return false
}
