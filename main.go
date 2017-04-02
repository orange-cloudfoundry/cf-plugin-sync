package main

import (
	"bytes"
	"code.cloudfoundry.org/cli/plugin"
	"github.com/urfave/cli"
	"fmt"
)

var version_major int = 1
var version_minor int = 0
var version_build int = 0
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
	finalArgs := append([]string{APP_NAME}, args...)
	app.Run(finalArgs)
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