#!/bin/bash

set -xe

export GIT_USER=$GITHUB_USER_NAME
export GIT_PASS=$GITHUB_PASSWORD

cd $WORKSPACE/hack/jenkins
if [ "$ENABLE_PUSH_TAGS_MODE" == "true" ]; then
	sudo -E python promote_build.py -b $branch -n $build_num -m $BUILD_PROMOTION_MODE --verbose --push-tags
else
	sudo -E python promote_build.py -b $branch -n $build_num -m $BUILD_PROMOTION_MODE --verbose
fi
ret_code=$?
if [ $ret_code -ne 0 ]; then
	echo "promote_build.py script exited with an error"
    exit 1
fi

