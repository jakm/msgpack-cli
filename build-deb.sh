#!/bin/bash

cd $(dirname $0)

gbp dch -R --urgency=low --debian-tag='%(version)s' --git-author
debuild -i -I
