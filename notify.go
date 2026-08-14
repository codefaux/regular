package main

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	mail "github.com/xhit/go-simple-mail/v2"
)

const (
	exitMessageText = "Exit message: %v\n\n"
	exitStatusText  = "Exit status: %v\n\n"
	failureSubject  = "Job %q failed"
	successSubject  = "Job %q succeeded"
)

type notifyMode string

const (
	notifyAlways    notifyMode = "always"
	notifyNever     notifyMode = "never"
	notifyOnFailure notifyMode = "on-failure"
)

type notifyWhenDone func(string, CompletedJob) error

func parseNotifyMode(mode string) (notifyMode, error) {
	switch mode {
	case string(notifyAlways):
		return notifyAlways, nil
	case string(notifyNever):
		return notifyNever, nil
	case string(notifyOnFailure), "":
		return notifyOnFailure, nil
	default:
		return "", fmt.Errorf("unknown notify mode: %v", mode)
	}
}

func notifyIfNeeded(notify notifyWhenDone, mode notifyMode, jobName string, completed CompletedJob) error {
	if mode == notifyNever {
		return nil
	}

	if !(mode == notifyAlways || mode == notifyOnFailure && !completed.IsSuccess()) {
		return nil
	}

	return notify(jobName, completed)
}

func notifyUserByEmail(db *appDB) notifyWhenDone {
	return func(jobName string, completed CompletedJob) error {
		subject, text, err := formatMessage(db, jobName, completed)
		if err != nil {
			return fmt.Errorf("failed to format notification message: %v", err)
		}

		creds := getCredentials(db.db)

		localhostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("failed to get current hostname: %v", err)
		}

		currentUser, err := user.Current()
		if err != nil {
			return fmt.Errorf("failed to get current user: %v", err)
		}

		client := mail.NewSMTPClient()
		client.Host = creds.Server
		client.Port = creds.Port
		client.Username = creds.User

		client.Password = GenerateCredential(localhostname, creds.Server, creds.User, creds.Port)

		mailer, err := client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: \n%v\n\nNOTE: %s now generates credentials unless overridden.\n"+
				"Generated credentials are determined uniquely via SMTP username, hostname, and port.\n"+
				"Used credentials (after overrides) were; user '%s' password '%s'\n"+
				"Create or modify an account to match, or specify your own.", err, "regular", client.Username, client.Password)
		}

		email := mail.NewMSG()
		email.SetFrom(localUserAddress(currentUser.Username, localhostname)).
			AddTo(creds.SendTo).
			SetSubject(subject).
			SetBody(mail.TextPlain, text)

		if completed.AttachLogs != AttachNever {
			stdout_log, stderr_log, err := getLogsForAttachment(db, jobName)
			if err == nil {
				if len(stdout_log.Data) > 1 {
					email.Attach(&stdout_log)
				}
				if len(stderr_log.Data) > 1 {
					email.Attach(&stderr_log)
				}
			} else {
				fmt.Printf("Error reading job logs for attachment: %s", jobName)
			}
		}

		if err := email.Send(mailer); err != nil {
			return fmt.Errorf("failed to send email: %v\n", err)
		}

		return nil
	}
}

func formatMessage(db *appDB, jobName string, completed CompletedJob) (string, string, error) {
	subjectTemplate := successSubject
	if !completed.IsSuccess() {
		subjectTemplate = failureSubject
	}
	subject := fmt.Sprintf(subjectTemplate, jobName)

	var sb strings.Builder
	if completed.ExitMessage != "" {
		sb.WriteString(fmt.Sprintf(exitMessageText, completed.ExitMessage))
	} else if completed.ExitCode != 0 {
		sb.WriteString(fmt.Sprintf(exitStatusText, completed.ExitCode))
	}

	if db != nil {
		for _, logName := range []string{"stdout", "stderr"} {
			lines, err := db.getJobLogs(jobName, logName, defaultLogLines)
			if err != nil {
				return "", "", fmt.Errorf("error reading log: %w", err)
			}

			if len(lines) == 0 {
				continue
			}

			sb.WriteString(logName)
			sb.WriteString(":\n")

			for _, line := range lines {
				sb.WriteString("> ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
	}

	return subject, sb.String(), nil
}

func getLogsForAttachment(db *appDB, jobName string) (mail.File, mail.File, error) {
	var lines_out, lines_err []string
	var err error

	lines_out, err = db.getJobLogs(jobName, "stdout", -1)
	if err != nil {
		return mail.File{}, mail.File{}, fmt.Errorf("error reading log: %w", err)
	}

	lines_err, err = db.getJobLogs(jobName, "stderr", -1)
	if err != nil {
		return mail.File{}, mail.File{}, fmt.Errorf("error reading log: %w", err)
	}

	return mail.File{
			Name:     "stdout.log",
			MimeType: "text/plain",
			Data:     []byte(strings.Join(lines_out, "\n") + "\n"),
		},
		mail.File{
			Name:     "stderr.log",
			MimeType: "text/plain",
			Data:     []byte(strings.Join(lines_err, "\n") + "\n"),
		},
		nil
}
