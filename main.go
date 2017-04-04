package main

import (
	"bytes"
	"code.cloudfoundry.org/cli/plugin"
	"gopkg.in/urfave/cli.v1"
	"fmt"
	"os"
)

var version_major int = 1
var version_minor int = 1
var version_build int = 1
var commandPluginHelpUsage = `   {{.HelpName}}cf sync{{if .Flags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{if .Description}}
DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}
`

const (
	APP_NAME = "sync"
)

type SyncPlugin struct {
	syncCommand *SyncCommand
}

func (c *SyncPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	c.syncCommand = &SyncCommand{cliConnection}
	app := cli.NewApp()
	app.Version = fmt.Sprintf("%d.%d.%d", version_major, version_minor, version_build)
	app.Usage = "Synchronize a folder to a container directory."
	app.Commands = generateCommand(c.syncCommand)
	app.ErrWriter = os.Stderr
	app.Writer = os.Stdout
	finalArgs := append([]string{APP_NAME}, args...)
	err := app.Run(finalArgs)
	if err != nil {
		logger.Error(err.Error())
	}
}
func (c *SyncPlugin) GetMetadata() plugin.PluginMetadata {

	pluginCommands := []plugin.Command{}
	commands := generateCommand(c.syncCommand)
	for _, command := range commands {
		bufferedHelp := new(bytes.Buffer)
		cli.HelpPrinter(bufferedHelp, commandPluginHelpUsage, command)
		pluginCommands = append(pluginCommands, plugin.Command{
			Name:     command.Name,
			HelpText: command.Usage,
			UsageDetails: plugin.Usage{
				Usage: bufferedHelp.String(),
			},
		})
	}
	return plugin.PluginMetadata{
		Name: APP_NAME,
		Version: plugin.VersionType{
			Major: version_major,
			Minor: version_minor,
			Build: version_build,
		},
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 7,
			Build: 0,
		},
		Commands: pluginCommands,
	}
}
func main() {
	plugin.Start(new(SyncPlugin))
}