def should_run(finished, timestamp, dow, **_):
    return True

# Random delay of up to 1 hour.
jitter = 0

# Kill the job if it runs longer than this.
# 0 (default) means no timeout.
timeout = one_hour

# Command to run.
command = [
    "sh",
    "-c",
    "echo ${REGULAR_JOB_DIR}/../common/docker-update-build.sh",
]

# Queue name (the default is the name of the job directory).
queue = "test"

# Write output to log files (default).
log = True

# When to send notifications: "always", "on-failure" (default), "never".
notify = "always"

# Allow multiple instances in queue (default).
duplicate = False

# Enable/disable the job (default).
enable = False

