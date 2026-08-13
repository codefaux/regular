package main

import (
	"fmt"
	"os"
	"os/user"
)

func coalesce[T comparable](a, b T) T {
	var zero_or_empty T
	if a != zero_or_empty {
		return a
	}
	return b
}

func (c *CredentialsCmd) Run(config Config) error {
	detectUser := "unknown"
	currentUser, err := user.Current()
	if err == nil {
		detectUser = currentUser.Username
	}

	detectHostname := "unknown"
	localHostname, err := os.Hostname()
	if err == nil {
		detectHostname = localHostname
	}

	fmt.Printf("Defaults:   hostname '%s'   user '%s'   server '%s'   port '%d'\n\n", detectHostname, detectUser, smtpServer, smtpPort)

	useHostname := coalesce(c.LocalHostname, detectHostname)
	useUser := coalesce(c.SMTPUsername, detectUser)
	useServer := coalesce(c.SMTPHostname, smtpServer)
	usePort := coalesce(c.SMTPPort, smtpPort)

	fmt.Printf("Generating:   hostname '%s'   user '%s'   server '%s'   port '%d'\n", useHostname, useUser, useServer, usePort)
	fmt.Printf("password: %s\n\n", GenerateCredential(useHostname, useUser, useServer, usePort))

	return nil
}
