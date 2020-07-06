#!/bin/bash

set -eu -o pipefail

GITHUB_REF=${GITHUB_REF:-unknown}
NO_SLASHES=${GITHUB_REF//\//-}
BRANCH=${NO_SLASHES/refs-heads-/}
RUN_NUMBER=${GITHUB_RUN_NUMBER:-0}
if [ "$BRANCH" = "master" ]; then
    echo "1.2.$RUN_NUMBER"
else
    echo "0.2.$RUN_NUMBER-$BRANCH"
fi
