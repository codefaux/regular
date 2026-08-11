#!/usr/bin/env bash

set -euo pipefail

APP_DIR="/docker/app"
REPO_DIR=""
USE_BRANCH=""
USE_TAG=""
TRACKING_FILE=""
REPO_BRANCH=""

# If we have a repo dir, begin repo update procedure
if [[ -z "$REPO_DIR" ]]; then
    echo "ERROR: REPO_DIR is empty. Skipping git update." >&2
else
    echo "==> Updating Git repository..."

    # If USE_BRANCH or USE_TAG are not EMPTY STRINGS
    if [[ -n "$USE_BRANCH" || -n "$USE_TAG" ]]; then
        cd "$REPO_DIR" || {
            echo "ERROR: Failed to change directory to '$REPO_DIR'. Aborted." >&2
            exit 1
        }

        # if we want to track latest repo tag
        if [[ -n "$USE_TAG" ]]; then
            # get new tags from repo
            git fetch --tags || {
                echo "ERROR: 'git fetch --tags' failed. Aborted." >&2
                exit 1
            }

            # pick latest tag
            TAG=$(git tag --sort=-v:refname | head -n 1)

            if [[ -z "$TAG" ]]; then
                echo "ERROR: No Git tags found. Aborted." >&2
                exit 1
            fi

            # if we have a tracking file and tags match, exit cleanly
            if [[ -f "$TRACKING_FILE" ]] && [[ "$(cat "$TRACKING_FILE")" == "$TAG" ]]; then
                echo "==> Already at latest tag '${TAG}'. Nothing to do."
                exit 0
            fi

            echo "==> New tag detected: '${TAG}'"

            # store new tag for tracking
            printf '%s\n' "$TAG" > "$TRACKING_FILE" || {
                echo "ERROR: Failed to write tag to '$TRACKING_FILE'. Aborted." >&2
                exit 1
            }

            # checkout detached as of tag
            git checkout --detach "$TAG" || {
                echo "ERROR: 'git checkout $TAG' failed. Aborted." >&2
                exit 1
            }

            echo "==> Repo switched to tag '${TAG}'"

        elif [[ -n "$USE_BRANCH" ]]; then
            # must also have branch set
            if [[ -z "$REPO_BRANCH" ]]; then
                echo "ERROR: REPO_BRANCH is empty. Aborted." >&2
                exit 1
            fi

            # get upstream branch
            git fetch origin "$REPO_BRANCH" || {
                echo "ERROR: Failed to fetch branch '$REPO_BRANCH'. Aborted." >&2
                exit 1
            }

            # get current commit on branch
            UPSTREAM_COMMIT=$(git rev-parse "origin/$REPO_BRANCH") || {
                echo "ERROR: Failed to resolve upstream branch 'origin/$REPO_BRANCH'. Aborted." >&2
                exit 1
            }

            # if we have a tracking file and commits match, exit cleanly
            if [[ -f "$TRACKING_FILE" ]] && [[ "$(cat "$TRACKING_FILE")" == "$UPSTREAM_COMMIT" ]]; then
                echo "==> Already at latest commit '${UPSTREAM_COMMIT}'. Nothing to do."
                exit 0
            fi

            echo "==> New commit detected on '${REPO_BRANCH}': '${UPSTREAM_COMMIT}'"

            # store new commit for tracking
            printf '%s\n' "$UPSTREAM_COMMIT" > "$TRACKING_FILE" || {
                echo "ERROR: Failed to write commit to '$TRACKING_FILE'. Aborted." >&2
                exit 1
            }

            # checkout upstream commit
            git checkout --detach "$UPSTREAM_COMMIT" || {
                echo "ERROR: 'git checkout $UPSTREAM_COMMIT' failed. Aborted." >&2
                exit 1
            }

            echo "==> Repo switched to upstream commit '${UPSTREAM_COMMIT}'"
        fi
    fi
fi

echo "==> Building Docker images..."
cd "$APP_DIR" || {
    echo "ERROR: Failed to change directory to '$APP_DIR'. Aborted." >&2
    exit 1
}

docker compose build || {
    echo "ERROR: Docker Compose build failed. Aborted." >&2
    exit 1
}

echo "==> Starting Docker Compose services..."
docker compose up -d || {
    echo "ERROR: Docker Compose failed to start." >&2
    exit 1
}

echo "==> Deployment completed successfully."
