package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"

	mail "github.com/xhit/go-simple-mail/v2"
)

const (
	smtpServer   = "127.0.0.1"
	smtpPort     = 25
	fromUsername = "regular"

	errorText      = "Error: %v\n\n"
	exitStatusText = "Exit status: %v\n\n"
	failureSubject = "Job %q failed"
	successSubject = "Job %q succeeded"
)

type notifyMode string

const (
	notifyAlways    notifyMode = "always"
	notifyNever     notifyMode = "never"
	notifyOnFailure notifyMode = "on-failure"
)

type notifyWhenDone func(string, CompletedJob) error

func GenerateCredential(hostname string, user string, host string, port int) string {
	input := "v1\x00" + hostname + "\x00" + user + "\x00" + host + "\x00" + strconv.Itoa(port)
	sum := sha256.Sum256([]byte(input))

	return base64.RawURLEncoding.EncodeToString(sum[:10])
}

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

func localUserAddress(username string) string {
	return username + "@localhost"
}

func notifyUserByEmail(db *appDB) notifyWhenDone {
	return func(jobName string, completed CompletedJob) error {
		subject, text, err := formatMessage(db, jobName, completed)
		if err != nil {
			return fmt.Errorf("failed to format notification message: %v", err)
		}

		localhostname, err := os.Hostname()
		if err != nil {
			return err
		}

		currentUser, err := user.Current()
		if err != nil {
			return fmt.Errorf("failed to get current user: %v", err)
		}

		host := os.Getenv("SMTP_HOST")
		if host == "" {
			host = smtpServer
		}

		port := smtpPort
		if value := os.Getenv("SMTP_PORT"); value != "" {
			port, err = strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid SMTP_PORT: %v", err)
			}
		}

		username := os.Getenv("SMTP_USERNAME")
		if username == "" {
			username = currentUser.Username
		}

		password := os.Getenv("SMTP_PASSWORD")

		client := mail.NewSMTPClient()
		client.Host = host
		client.Port = port

		client.Username = username

		if password == "" {
			client.Password = GenerateCredential(localhostname, username, host, port)
		}

		mailer, err := client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: \n%v\n\nNOTE: %s now generates credentials unless overridden.\n"+
				"Generated credentials are determined uniquely via SMTP username, hostname, and port.\n"+
				"Used credentials (after overrides) were; user '%s' password '%s'\n"+
				"Create or modify an account to match, or specify your own.", err, "regular", client.Username, client.Password)
		}

		email := mail.NewMSG()
		email.SetFrom(localUserAddress(fromUsername)).
			AddTo(localUserAddress(currentUser.Username)).
			SetSubject(subject).
			SetBody(mail.TextPlain, text)

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
	if completed.Error != "" {
		sb.WriteString(fmt.Sprintf(errorText, completed.Error))
	} else if completed.ExitStatus != 0 {
		sb.WriteString(fmt.Sprintf(exitStatusText, completed.ExitStatus))
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
