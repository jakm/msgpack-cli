#!/bin/bash

cd $(dirname $0)

NEW_VERSION=$(awk 'match($0, /^const __VERSION__ = "([0-9]+)\.([0-9]+)"$/, arr) { arr[2] = arr[2] + 1; printf("%d.%d", arr[1], arr[2]) }' version.go)
sed -i "s/^const __VERSION__ = .*/const __VERSION__ = \"$NEW_VERSION\"/g" version.go

gbp dch -R --urgency=low --debian-tag='%(version)s' --git-author --new-version=$NEW_VERSION
debuild -i -I

git add version.go debian/changelog
git commit -m "Version $NEW_VERSION"
git tag -m "" $NEW_VERSION
