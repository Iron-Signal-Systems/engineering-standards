package model

import "time"

type Status string

const (
	Pass Status = "PASS"
	Fail Status = "FAIL"
	Warn Status = "WARN"
	Info Status = "INFO"
)

type Action struct {
	Label       string
	Description string
	Command     string
}

type Check struct {
	Section  string
	Name     string
	Status   Status
	Detail   string
	LogPath  string
	Actions  []Action
	Started  time.Time
	Finished time.Time
}

type Summary struct {
	Checks []Check
}

func (s Summary) Failed() bool {
	for _, check := range s.Checks {
		if check.Status == Fail {
			return true
		}
	}
	return false
}

func (s Summary) Count(status Status) int {
	count := 0
	for _, check := range s.Checks {
		if check.Status == status {
			count++
		}
	}
	return count
}
