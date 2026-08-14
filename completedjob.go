package main

import (
	"time"
)

type CompletedJob struct {
	ExitMessage   string
	ExitCode      int
	ErrorIsCode   *int
	WarningIsCode *int
	SuccessIsCode *int
	Started       time.Time
	Finished      time.Time
	WasForced     bool
	AttachLogs    AttachMode
}

func (cj CompletedJob) ConsiderFailed() bool {
	if (cj.SuccessIsCode != nil) && (cj.ExitCode != *cj.SuccessIsCode) {
		return true
	}

	if (cj.ErrorIsCode != nil) && (cj.ExitCode == *cj.ErrorIsCode) {
		return true
	}

	if (cj.SuccessIsCode == nil) && (cj.ErrorIsCode == nil) {
		return cj.ExitCode == 0 && cj.ExitMessage == ""
	} else {
		return false
	}
}
