package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
)

func coalesce[T comparable](values ...T) T {
	var zero_or_empty T

	for _, value := range values {
		if value != zero_or_empty {
			return value
		}
	}

	return zero_or_empty
}

func localUserAddress(username string, hostname string) string {
	return username + "@" + hostname
}

func GenerateCredential(hostname string, user string, server string, port int) string {
	input := "v1\x00" + hostname + "\x00" + user + "\x00" + server + "\x00" + strconv.Itoa(port)
	sum := sha256.Sum256([]byte(input))

	return base64.RawURLEncoding.EncodeToString(sum[:10])
}

func (c *CredentialsCmd) Run(config Config) error {
	detectHostname := "unknown"
	localHostname, err := os.Hostname()
	if err == nil {
		detectHostname = localHostname
	}

	db, err := openAppDB(config.StateRoot)
	if err != nil {
		return err
	}
	defer db.close()

	creds := getCredentials(db.db)

	useHostname := coalesce(c.LocalHostname, detectHostname)
	useUser := coalesce(c.User, creds.User)
	useServer := coalesce(c.Server, creds.Server)
	usePort := coalesce(c.Port, creds.Port)
	useSendTo := coalesce(c.SendTo, creds.SendTo)

	fmt.Printf("Generating:   hostname '%s'   user '%s'   server '%s'   port '%d'   to '%s'\n", useHostname, useUser, useServer, usePort, useSendTo)
	fmt.Printf("password: %s\n\n", GenerateCredential(useHostname, useUser, useServer, usePort))

	if c.Store {
		err := db.saveCredentials(creds)
		if err != nil {
			return fmt.Errorf("Error writing credentials to database: %v", err)
		}

	}

	return nil
}
