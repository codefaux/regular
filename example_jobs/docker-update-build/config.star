def should_run(finished, timestamp, dow, **_):
    return timestamp - finished >= 7 * one_day

# Random delay of up to 1 hour.
jitter = one_hour

# Kill the job if it runs longer than this.
# 0 (default) means no timeout.
timeout = 4 * one_hour

# Command to run.
command = [
    "sh",
    "-c",
    "${REGULAR_JOB_DIR}/docker-update-build.sh",
]

# Queue name (the default is the name of the job directory).
queue = "docker"

# Write output to log files (default).
log = True

# When to send notifications: "always", "on-failure" (default), "never".
notify = "always"

# Allow multiple instances in queue (default).
duplicate = False

# Enable/disable the job (default).
enable = False

