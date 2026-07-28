# Regular

> [!WARNING]
> **This project is in early development.**\
> It is not ready for others to use.
> I (@codefaux) have made simple hacks to tweak this to my needs.
> Most code and all ownership/copyright/authorship references are unchanged.
> Much of the documentation may refer to user-specific service/installation
> and is incorrect.

**Regular** is a job scheduler like [cron](https://en.wikipedia.org/wiki/Cron) and [anacron](https://en.wikipedia.org/wiki/Anacron).

## Features

- Configuration using [Starlark](https://laurent.le-brun.eu/blog/an-overview-of-starlark), a small configuration language based on Python.
  You can use expressions like `hour in [9, 18] and minute == 0` to define when a job will run.
- Flexible scheduling based on current time and the last completed job
- Jitter to mitigate the [thundering herd problem](https://en.wikipedia.org/wiki/Thundering_herd_problem)
- Job queues to configure what jobs run sequentially and in parallel
- Built-in logging and status reporting
- Built-in email notifications on localhost
- Hot reloading of job configuration on file change

## Installation

You will need Go 1.22 or later.

Pull the repo, run systemd/install.sh

## Configuration

Jobs are defined in Starlark files named `config.star` in subdirectories of the config directory.
For example:

```starlark
# Run if at least a day has passed since the last run
# and it isn't the weekend.
def should_run(finished, timestamp, dow, **_):
    return dow not in [0, 6] and timestamp - finished >= one_day

# Random delay of up to 1 hour.
jitter = one_hour

# Kill the job if it runs longer than this.
# 0 (default) means no timeout.
timeout = one_hour

# Command to run.
command = [
    "sh",
    "-c",
    "backup.sh ~/docs /backup/docs",
]

# Queue name (the default is the name of the job directory).
queue = "backup"

# Write output to log files (default).
log = True

# When to send notifications: "always", "on-failure" (default), "never".
notify = "always"

# Allow multiple instances in queue (default).
duplicate = False

# Enable/disable the job (default).
enable = True
```

Each job directory can also have an optional `job.env` file with environment variables:

```
PATH=${PATH}:${HOME}/bin
BACKUP_OPTS=--compress
```

## Usage

### General

- **regular** [_flags_] _command_
  - **-h**, **--help** Print help
  - **-V**, **--version** Print version number and exit
  - **-c**, **--config-dir** Path to config directory
  - **-s**, **--state-dir** Path to state directory

### Commands

Start the scheduler:

- **regular start**

Run specific jobs once:

- **regular run** [**--force**] [_job-names_...]

> [!NOTE]
> When a `regular start` daemon is running, `run` connects to it over a Unix socket and streams the job's stdout, stderr, and exit code back to your terminal.
> Ad hoc runs use the same queues and avoid duplication just like scheduled runs.
>
> With no daemon running, `run` falls back to executing the job in its own process.
> Concurrent standalone invocations avoid conflict using a lock file in the state directory.

Check job status:

- **regular status** [**-l** _lines_] [_job-names_...]

View application log:

- **regular log** [**-l** _lines_]

List available jobs:

- **regular list**

## File locations

Default paths (override with **-c** and **-s**):

- Config: `/etc/regular/`
  - Global environment: `/etc/regular/global.env`
  - Job config: `/etc/regular/<job>/config.star`
  - Job environment: `/etc/regular/<job>/job.env`
  - Job executable (script): `/etc/regular/<job>/job`

- State: `/var/lib/regular/regular/`
  - App log: `/var/lib/regular/regular/app.log`
  - Database: `/var/lib/regular/regular/state.sqlite3`
  - Lock file: `/var/lib/regular/regular/app.lock`.
    When in use, this file prevents multiple instances of `regular start` from running at the same time.
    `regular run` also takes this lock when no daemon is running.
  - Logs for the latest job: `/var/lib/regular/regular/<job>/{stdout,stderr}.log`.
    These logs and earlier logs are also stored in the database.

- Daemon socket: `/var/run/regular/socket` or `/run/regular/socket` (or, with no runtime dir, a subdir under `$TMPDIR`).
  Set `REGULAR_SOCK` to override.
  The socket is created with mode `0600` and the client refuses to connect to one owned by another user.

The config and state directory are created automatically when you run `regular start` or `regular run`.

Job logs are truncated at 256 KiB.
There is currently no built-in way to remove old logs from the database.
You can use the [**sqlite3** command shell](https://www.sqlite.org/cli.html) to remove logs manually.

All files and directories are created with 0600 and 0700 permissions respectively.

## systemd service

Regular's repository includes a systemd unit file for running the scheduler automatically.

To install and enable the service, clone the repository, then run:

```shell
cd systemd/
# Run this as your user, not as root.
./install.sh
```

This will:

- Build the binary using go
- Install the binary to /usr/bin/regular
- Create a service file in `/etc/systemd/system/`
- Enable the service to start automatically
- Start the service immediately

To check the service status:

```shell
systemctl status regular
```

To view logs:

```shell
journalctl -u regular -f
```

## Shell completions

Regular includes shell completions for the fish shell.

### fish shell

To install completions for the [fish shell](https://en.wikipedia.org/wiki/Fish_(Unix_shell)), clone the repository, then run:

```shell
cd completions/
./install.fish
```

This will copy the completion file to your fish configuration directory.

## License

MIT.
See the file [`LICENSE`](LICENSE).
