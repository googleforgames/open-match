#!/bin/bash
# Usage:
# ./release.sh 0.5.0-82d034f unstable
# ./release.sh [SOURCE VERSION] [DEST VERSION]

# This is a basic shell script to publish the latest Open Match Images
# There's no guardrails yet so use with care.
# Purge Images
# docker rmi $(docker images -a -q)
# 0.4.0-82d034f
SOURCE_VERSION=$1
DEST_VERSION=$2
SOURCE_PROJECT_ID=open-match-build
DEST_PROJECT_ID=open-match-public-images
IMAGE_NAMES="openmatch-backend openmatch-frontend openmatch-query openmatch-synchronizer openmatch-minimatch openmatch-demo-first-match openmatch-mmf-go-soloduel openmatch-mmf-go-pool openmatch-evaluator-go-simple openmatch-swaggerui openmatch-reaper"

for name in $IMAGE_NAMES
do
    source_image=gcr.io/$SOURCE_PROJECT_ID/$name:$SOURCE_VERSION
    dest_image=gcr.io/$DEST_PROJECT_ID/$name:$DEST_VERSION
    docker pull $source_image
    docker tag $source_image $dest_image
    docker push $dest_image
done

echo "=============================================================="
echo "=============================================================="
echo "=============================================================="
echo "=============================================================="

echo "Add these lines to your release notes:"
for name in $IMAGE_NAMES
do
    echo "docker pull gcr.io/$DEST_PROJECT_ID/$name:$DEST_VERSION"
done
