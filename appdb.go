package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	_ "modernc.org/sqlite"
)

type appDB struct {
	db *sql.DB
}

type credentials struct {
	Server string
	User   string
	Port   int
	SendTo string
}

const v0_schema string = `
				CREATE TABLE IF NOT EXISTS completed_jobs (
					id INTEGER PRIMARY KEY,
					job_name TEXT NOT NULL,
					error TEXT,
					exit_status INTEGER NOT NULL,
					started DATETIME NOT NULL,
					finished DATETIME NOT NULL,
					created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
				);

				CREATE INDEX IF NOT EXISTS idx_completed_jobs_job_name
					ON completed_jobs(job_name);

				CREATE TABLE IF NOT EXISTS job_logs (
					id INTEGER PRIMARY KEY,
					completed_job_id INTEGER NOT NULL,
					log_name TEXT NOT NULL,
					line_number INTEGER NOT NULL,
					line TEXT NOT NULL,
					FOREIGN KEY(completed_job_id) REFERENCES completed_jobs(id)
				);

				CREATE INDEX IF NOT EXISTS idx_job_logs_completed_job_id ON job_logs(completed_job_id);
`
const v1_schema string = v0_schema + `
				CREATE TABLE IF NOT EXISTS credentials (
					id INTEGER PRIMARY KEY CHECK (id = 1),
					server TEXT NOT NULL,
					user TEXT NOT NULL,
					port INTEGER NOT NULL,
					sendto TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS database_version (
					id INTEGER PRIMARY KEY CHECK (id = 1),
					version INTEGER NOT NULL
				);
`

var schemas = map[int]string{
	0: v0_schema,
	1: v1_schema,
}

func openAppDB(stateRoot string) (*appDB, error) {
	if err := os.MkdirAll(stateRoot, dirPerms); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %v", err)
	}

	dbPath := filepath.Join(stateRoot, appDBFileName)
	dsnPath := "file:" + dbPath + "?_pragma=foreign_keys(1)"

	_, err := os.Stat(dbPath)
	dbExists := err == nil

	db, err := sql.Open("sqlite", dsnPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if dbExists {
		if err := checkSchema(db); err != nil {
			db.Close()
			return nil, err
		}
	} else {
		if err := createSchema(db, -1); err != nil {
			db.Close()
			return nil, err
		}
	}

	return &appDB{db: db}, nil
}

func (c *appDB) close() error {
	return c.db.Close()
}

func checkSchema(db *sql.DB) error {
	var detectVersion int

	var v0_or_greater bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM sqlite_master
			WHERE type = 'table' AND name = 'job_logs'
		)
	`).Scan(&v0_or_greater)
	if (err != nil) || (!v0_or_greater) {
		return fmt.Errorf("loaded file does not appear to be for regular.")
	}

	var v1_or_greater bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM sqlite_master
			WHERE type = 'table' AND name = 'database_version'
		)
	`).Scan(&v1_or_greater)
	if (err != nil) || (!v1_or_greater) {
		return upgradeSchema(db, 0, dbVersion)
	}

	err = db.QueryRow(`
		SELECT version
		FROM database_version
		WHERE id = 1
	`).Scan(&detectVersion)
	if err != nil {
		return fmt.Errorf("failed to read database version: %w", err)
	}

	if detectVersion > dbVersion {
		return fmt.Errorf(
			"database version %d is newer than supported version %d",
			detectVersion,
			dbVersion,
		)
	}

	if detectVersion < dbVersion {
		return upgradeSchema(db, detectVersion, dbVersion)
	}

	return nil
}

func mark_db_version(tx *sql.Tx, version int) error {
	_, err := tx.Exec(`
		INSERT INTO database_version (id, version)
		VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET version = excluded.version
	`, strconv.Itoa(version))
	if err != nil {
		return fmt.Errorf("failed to update database version: %w", err)
	}

	return nil
}

func upgradeSchema(db *sql.DB, from, to int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin schema upgrade: %w", err)
	}
	defer tx.Rollback()

	for version := from + 1; version <= to; version++ {
		switch version {
		case 1:
			// added smtp credentials storage
			_, err = tx.Exec(schemas[1])
		}

		if err != nil {
			return fmt.Errorf("failed to upgrade database to version %d: %w", version, err)
		}
	}

	mark_db_version(tx, to)

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit schema upgrade: %w", err)
	}

	return nil
}

func createSchema(db *sql.DB, version int) error {
	if version == -1 {
		version = dbVersion
	}

	schema, ok := schemas[version]
	if !ok {
		return fmt.Errorf("schema version %d not found", version)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin schema upgrade: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema version %d: %w", version, err)
	}

	err = mark_db_version(tx, version)
	if err != nil {
		return fmt.Errorf("failed to mark database version %d: %w", version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit schema upgrade: %w", err)
	}

	return nil
}

func (c *appDB) saveCredentials(creds *credentials) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.Exec(`
		INSERT INTO credentials (id, server, user, port, sendto)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			server = excluded.server,
			user = excluded.user,
			port = excluded.port,
			sendto = excluded.sendto
	`, creds.Server, creds.User, creds.Port, creds.SendTo)

	if err != nil {
		return err
	}

	_, err = result.LastInsertId()
	if err != nil {
		return err
	}

	return tx.Commit()
}

func getCredentials(db *sql.DB) *credentials {
	var creds credentials

	err := db.QueryRow(`
		SELECT server, user, port, sendto
		FROM credentials
		WHERE id = 1
	`).Scan(
		&creds.Server,
		&creds.User,
		&creds.Port,
		&creds.SendTo,
	)
	if err != nil {

		localhostname, err := os.Hostname()
		if err != nil {
			localhostname = "localhost"
		}

		currentUsername := "unknown"
		currentUser, err := user.Current()
		if err == nil {
			currentUsername = currentUser.Username
		}

		creds.Server = localhostname
		creds.User = currentUsername
		creds.Port = 25
		creds.SendTo = localUserAddress(currentUsername, localhostname)

		return &creds
	}

	return &creds
}

func (c *appDB) saveCompletedJob(jobName string, completed CompletedJob, logs []logFile) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.Exec(`
		INSERT INTO completed_jobs (
			job_name,
			error,
			exit_status,
			started,
			finished
		) VALUES (?, ?, ?, ?, ?)`,
		jobName,
		completed.ExitMessage,
		completed.ExitCode,
		completed.Started,
		completed.Finished,
	)
	if err != nil {
		return err
	}

	jobID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	for _, logFile := range logs {
		if err := c.saveLogFile(tx, jobID, logFile.name, logFile.path); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (c *appDB) saveLogFile(tx *sql.Tx, jobID int64, logName, path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	buf := make([]byte, maxLogBufferSize)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}
	buf = buf[:n]

	lineNum := 1
	scanner := bufio.NewScanner(bytes.NewReader(buf))
	// Allow a single line to be as long as the entire log buffer; otherwise
	// the default 64 KiB cap would fail the whole transaction on long lines.
	scanner.Buffer(make([]byte, 0, 64*1024), maxLogBufferSize)
	for scanner.Scan() {
		_, err = tx.Exec(`
			INSERT INTO job_logs (
				completed_job_id,
				log_name,
				line_number,
				line
			) VALUES (?, ?, ?, ?)`,
			jobID,
			logName,
			lineNum,
			scanner.Text(),
		)
		if err != nil {
			return err
		}
		lineNum++
	}
	return scanner.Err()
}

func (c *appDB) getLastCompleted(jobName string) (*CompletedJob, error) {
	var completed CompletedJob
	err := c.db.QueryRow(`
		SELECT
			error,
			exit_status,
			started,
			finished
		FROM completed_jobs
		WHERE job_name = ?
		ORDER BY id DESC LIMIT 1`,
		jobName,
	).Scan(
		&completed.ExitMessage,
		&completed.ExitCode,
		&completed.Started,
		&completed.Finished,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &completed, nil
}

func (c *appDB) getJobLogs(jobName string, logName string, limit int) ([]string, error) {
	rows, err := c.db.Query(`
		SELECT line
		FROM (
			SELECT l.line, l.line_number
			FROM job_logs l
			JOIN completed_jobs j ON j.id = l.completed_job_id
			WHERE l.log_name = ?
			AND j.id = (
				SELECT id
				FROM completed_jobs
				WHERE job_name = ?
				ORDER BY id DESC
				LIMIT 1
			)
			ORDER BY l.line_number DESC
			LIMIT ?
		)
		ORDER BY line_number ASC`,
		logName,
		jobName,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return nil, err
		}

		lines = append(lines, line)
	}

	return lines, rows.Err()
}
