#!/usr/bin/env bash
set -euo pipefail

TRACKING_FILE=${REGULAR_JOB_DIR}/version_tracking

# If we have a repo dir, begin repo update procedure
if [[ -n "$REPO_DIR" ]]; then
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
            if [[ -s "$TRACKING_FILE" ]] && [[ "$(cat "$TRACKING_FILE")" == "$TAG" ]]; then
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
            if [[ -s "$TRACKING_FILE" ]] && [[ "$(cat "$TRACKING_FILE")" == "$UPSTREAM_COMMIT" ]]; then
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
elif [[ -n "$USE_BUILDER" ]]; then
    # Check builder image
    if [[ -z "$BUILDER_IMAGE" ]]; then
        echo "ERROR: BUILDER_IMAGE is empty. Aborted." >&2
        exit 1
    fi

    if [[ -z "$TRACKING_FILE" ]]; then
        echo "ERROR: TRACKING_FILE is empty. Aborted." >&2
        exit 1
    fi

    echo "==> Checking builder image '${BUILDER_IMAGE}'..."

    # Pull the builder image so that we are checking the current image.
    docker pull "$BUILDER_IMAGE" || {
        echo "ERROR: Failed to pull builder image '${BUILDER_IMAGE}'. Aborted." >&2
        exit 1
    }

    # Get the immutable local image ID after the pull.
    BUILDER_ID=$(docker image inspect "$BUILDER_IMAGE" --format '{{.Id}}') || {
        echo "ERROR: Failed to identify builder image '${BUILDER_IMAGE}'. Aborted." >&2
        exit 1
    }

    # If we have a tracking file and the builder image matches, exit cleanly.
    if [[ -f "$TRACKING_FILE" ]] && [[ "$(cat "$TRACKING_FILE")" == "$BUILDER_ID" ]]; then
        echo "==> Builder image '${BUILDER_IMAGE}' is unchanged. Nothing to do."
        exit 0
    fi

    echo "==> New builder image detected: '${BUILDER_ID}'"

    # Store the builder image ID for tracking.
    printf '%s\n' "$BUILDER_ID" > "$TRACKING_FILE" || {
        echo "ERROR: Failed to write builder image ID to '$TRACKING_FILE'. Aborted." >&2
        exit 1
    }

    echo "==> Builder image tracking updated."
fi

cd "$APP_DIR" || {
    echo "ERROR: Failed to change directory to '$APP_DIR'. Aborted." >&2
    exit 1
}

if [[ -n "$USE_COMPOSE_BUILD" ]]; then
    echo "==> Building Docker images..."
    docker compose build ${COMPOSE_BUILD_NAME} || {
        echo "ERROR: Docker Compose build failed. Aborted." >&2
        exit 1
    }
fi

if [[ -n "$USE_COMPOSE_PULL" ]]; then
    echo "==> Pulling Docker images..."
    docker compose pull ${COMPOSE_PULL_NAME} || {
        echo "ERROR: Docker Compose build failed. Aborted." >&2
        exit 1
    }
fi

echo "==> Starting Docker Compose services..."
docker compose up -d || {
    echo "ERROR: Docker Compose failed to start." >&2
    exit 1
}

echo "==> Deployment completed successfully."
