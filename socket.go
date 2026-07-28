package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// defaultSocketPath returns the path where the daemon listens by default.
// REGULAR_SOCK overrides everything. Otherwise the path is built from
// $XDG_RUNTIME_DIR or one of the well-known per-user runtime directories,
// falling back to a per-user subdir under os.TempDir().
func defaultSocketPath() (string, error) {
	if envPath := os.Getenv(socketEnv); envPath != "" {
		return envPath, nil
	}

	candidates := []string{}

	candidates = append(
		candidates,
		"/run",
		"/var/run",
	)

	var runtimeDir string
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			runtimeDir = candidate
			break
		}
	}

	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}

	return filepath.Join(runtimeDir, appSocketFileName), nil
}

// checkSocketSecurity verifies a socket file is a Unix domain socket owned
// by the current user with mode 0600. Run on every client connection so we
// don't talk to a socket dropped in the runtime dir by someone else.
func checkSocketSecurity(socket string) error {
	info, err := os.Stat(socket)
	if err != nil {
		return fmt.Errorf("failed to stat socket: %w", err)
	}

	if (info.Mode() & os.ModeSocket) == 0 {
		return errors.New("path is not a Unix domain socket")
	}

	if info.Mode().Perm() != filePerms {
		return fmt.Errorf("incorrect socket permissions: %v", info.Mode().Perm())
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("failed to get socket system info")
	}

	if int(stat.Uid) != os.Getuid() {
		return fmt.Errorf("socket owned by wrong user: %d", stat.Uid)
	}

	return nil
}
