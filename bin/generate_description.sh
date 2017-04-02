#!/usr/bin/env bash
CURRENTDIR=`pwd`
melody_dir="$CURRENTDIR/bin/tools"
melody_file="$melody_dir/melody.phar"
composer_file="$CURRENTDIR/composer.phar"
if [ ! -f $melody_file ]
then
    echo "Melody not found."
    echo "Downloading melody ..."
    mkdir -p $melody_dir
    curl http://get.sensiolabs.org/melody.phar -o $melody_file
    echo "Downloading composer ..."
    curl https://getcomposer.org/composer.phar -o $composer_file
fi

if [ ! -f "$CURRENTDIR/out/cf-plugin-sync_linux_amd64" ]
then
    echo "Need to build every versions."
    $CURRENTDIR/bin/build-all.sh
fi

php "$melody_file" run "$CURRENTDIR/bin/scripts/generatePluginDescription.php"
cat repo-index.yml