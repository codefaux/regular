package main

import (
	"time"
)

type CompletedJob struct {
	ExitMessage string
	ExitCode    int
	Started     time.Time
	Finished    time.Time
}

func (cj CompletedJob) IsSuccess() bool {
	return cj.ExitCode == 0 && cj.ExitMessage == ""
}
