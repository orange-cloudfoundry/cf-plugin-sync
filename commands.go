package main

import "gopkg.in/urfave/cli.v1"

func generateCommand(c *SyncCommand) []cli.Command {

	return []cli.Command{
		{
			Name:      "sync",
			Usage:     "Synchronize a folder to a container directory.",
			ArgsUsage: "<app name>",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "source, s",
					Usage: "Source directory to sync file from container, if empty it will populated with data from container.",
				},
				cli.StringFlag{
					Name: "target, t",
					Usage: "Directory which will be sync from container",
				},
			},
			Description: "Synchronize a folder to a container directory by default a sync-appname folder will be created in current dir and target dir will be set to ~/app",
			Action: c.Sync,
		},
	}
}
