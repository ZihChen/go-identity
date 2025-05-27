#!/bin/bash

set -e

# 設置默認變量
APP_IMAGE_NAME=${APP_IMAGE_NAME:-"ms-identity-cat-web"}
WORKER_IMAGE_NAME=${WORKER_IMAGE_NAME:-"ms-identity-cat-worker"}
CONSUMER_IMAGE_NAME=${CONSUMER_IMAGE_NAME:-"ms-identity-cat-consumer"}
CI_COMMIT_SHA=${CI_COMMIT_SHA:-$(git rev-parse HEAD)}

# 構建 web app 映像
echo "構建 web app 映像..."
docker build \
  --build-arg CI_COMMIT_SHA=$CI_COMMIT_SHA \
  -t $APP_IMAGE_NAME:latest \
  -t $APP_IMAGE_NAME:$CI_COMMIT_SHA \
  "$@" \
  .

# 構建 worker 映像
echo "構建 worker 映像..."
docker build \
  --build-arg CI_COMMIT_SHA=$CI_COMMIT_SHA \
  -t $WORKER_IMAGE_NAME:latest \
  -t $WORKER_IMAGE_NAME:$CI_COMMIT_SHA \
  -f Dockerfile.worker \
  "$@" \
  .

# 構建 consumer 映像
echo "構建 consumer 映像..."
docker build \
  --build-arg CI_COMMIT_SHA=$CI_COMMIT_SHA \
  -t $CONSUMER_IMAGE_NAME:latest \
  -t $CONSUMER_IMAGE_NAME:$CI_COMMIT_SHA \
  -f Dockerfile.consumer \
  "$@" \
  .

echo "構建完成!"
echo "Web Image: $APP_IMAGE_NAME:$CI_COMMIT_SHA"
echo "Worker Image: $WORKER_IMAGE_NAME:$CI_COMMIT_SHA"
echo "Consumer Image: $CONSUMER_IMAGE_NAME:$CI_COMMIT_SHA"