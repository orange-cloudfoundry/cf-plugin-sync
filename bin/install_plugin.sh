#!/usr/bin/env bash
CURRENTDIR=`pwd`
PLUGIN_PATH="$CURRENTDIR/out/cf-plugin-sync"

$CURRENTDIR/bin/build
cf uninstall-plugin sync
cf install-plugin "$PLUGIN_PATH" -f