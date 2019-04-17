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
IMAGE_NAMES="openmatch-backendapi openmatch-frontendapi openmatch-mmforc openmatch-mmlogicapi openmatch-evaluator-simple openmatch-mmf-cs-mmlogic-simple openmatch-mmf-go-mmlogic-simple openmatch-mmf-go-grpc-serving-simple openmatch-mmf-py3-mmlogic-simple openmatch-backendclient openmatch-clientloadgen openmatch-frontendclient"

for name in $IMAGE_NAMES
do
    source_image=gcr.io/$SOURCE_PROJECT_ID/$name:$SOURCE_VERSION
    dest_image=gcr.io/$DEST_PROJECT_ID/$name:$DEST_PROJECT_ID
    dest_image=gcr.io/$DEST_PROJECT_ID/$name:$DEST_VERSION
    dest_image_latest=gcr.io/$DEST_PROJECT_ID/$name:latest
    docker pull $source_image
    docker tag $source_image $dest_image
    docker tag $source_image $dest_image_latest
    docker push $dest_image
    docker push $dest_image_latest
done
