#!/usr/bin/env bash

set -eu pipefail

# Queries unique to queries.sql are filtered.
comm -23 \
  <(grep -oP 'name: [A-Za-z]+' queries.sql | cut -d' '  -f 2 | sort -u) \
  <(grep -RhoP 'app.queries.[A-Za-z]+' . 2>/dev/null | cut -d. -f3 | sort -u)
