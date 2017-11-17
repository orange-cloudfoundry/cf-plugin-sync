# cf-plugin-sync

This is a Cloud Foundry cli plugin which gives the possibility to synchronize a local folder to a remote folder inside a 
Cloud Foundry app.
 
This way you only need to modify files inside your local folder to see changes in your app.

**Note**: 
- This plugin was made only for developing purpose, do not use on an app in production (ssh should be disabled in this context)
- If you run multiple instances of your app only the first instance (index 0) will be altered.

## Installation

### Install from plugin repository (recommended)
NOTE: This installation method requires that your client computer has access to the internet.
If internet access is not available from client computer use the manual method.

Verify you have a repo named `CF-Community` registered in your cf client.

```
cf list-plugin-repos
```
If the above command does not show `CF-Community` you can add the repo via:

```
cf add-plugin-repo CF-Community http://plugins.cloudfoundry.org/
```
Now that we have the cloud foundry community repo registered, install `sync`:

```
cf install-plugin -r CF-Community "sync"
```

### Installation from release binaries

1. Download latest release made for your os here: https://github.com/orange-cloudfoundry/cf-plugin-sync/releases
2. run `cf install-plugin path/to/previous/binary/downloaded`

## Usage

```
NAME:
   sync - Synchronize a folder to a container directory.

USAGE:
   cf sync [command options] <app name>
DESCRIPTION:
   Synchronize a folder to a container directory by default a sync-appname folder will be created in current dir and target dir will be set to ~/app

OPTIONS:
   --source value, -s value  Source directory to sync file from container, if empty it will populated with data from container.
   --target value, -t value  Directory which will be sync from container.
   --force-sync, -f          Resynchronize files from remote to source even if source folder is not empty.
```

## .syncignore

you can ignore files and directories from remote app by adding a `.syncignore` in the syle of a `.gitignore` file in source folder (or working directory). 

Example for php-buildpack:

`.syncignore` file:
```
/.*
/httpd
/php
```

## Tips

- If no source folder is passed, the plugin will create a folder named `sync-appname`
- Root folder inside app is `~/app`
- If the source folder is not empty, data will not be resynchronized

