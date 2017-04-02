package main

import (
	"code.cloudfoundry.org/cli/plugin"
	"encoding/json"
	"strings"
	"time"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/ssh/options"
	"os"
	"github.com/urfave/cli"
	"errors"
)

const (
	DEFAULT_SYNC_FOLDER = "sync"
	DEFAULT_ROOT_TARGET_FOLDER = "~/app"
)

type SyncCommand struct {
	cliConnection plugin.CliConnection
}

type SshInfo struct {
	AppSSHEndpoint           string `json:"app_ssh_endpoint"`
	AppSSHHostKeyFingerprint string `json:"app_ssh_host_key_fingerprint"`
}

func (s SyncCommand) getSourceDir(c *cli.Context, appName string) string {
	sourceDir := c.String("source")
	if sourceDir == "" {
		sourceDir = "./" + DEFAULT_SYNC_FOLDER + "-" + appName
	}
	return sourceDir
}
func (s SyncCommand) getTargetDir(c *cli.Context) string {
	targetDir := c.String("target")
	if targetDir == "" {
		return DEFAULT_ROOT_TARGET_FOLDER
	}
	if !strings.HasPrefix(targetDir, "/") {
		targetDir = "/" + targetDir
	}
	return DEFAULT_ROOT_TARGET_FOLDER + targetDir
}
func (s *SyncCommand) Sync(c *cli.Context) error {
	appName := c.Args().First()
	if appName == "" {
		return errors.New("You must pass an app name.")
	}
	sourceDir := s.getSourceDir(c, appName)
	targetDir := s.getTargetDir(c)
	logger.Info("Retrieving information about your app ...")
	data, err := s.cliConnection.CliCommandWithoutTerminalOutput("curl", "/v2/info")
	if err != nil {
		return err
	}
	var sshInfo SshInfo
	err = json.Unmarshal([]byte(strings.Join(data, "")), &sshInfo)
	if err != nil {
		return err
	}
	app, err := s.cliConnection.GetApp(appName)
	if err != nil {
		return err
	}
	sslDisabled, err := s.cliConnection.IsSSLDisabled()
	if err != nil {
		return err
	}
	logger.Info("Finished retrieving information about your app.\n")
	logger.Info("Authenticating for ssh ...")
	data, err = s.cliConnection.CliCommandWithoutTerminalOutput("ssh-code")
	if err != nil {
		return err
	}
	keepAliveInterval := 30 * time.Second
	token := data[0]
	secureShell := NewSecureShell(
		DefaultSecureDialer(),
		DefaultListenerFactory(),
		keepAliveInterval,
		models.Application{
			ApplicationFields: models.ApplicationFields{
				//must be diego to have ssh works
				Diego: app.Diego,
				// guid can be found by doing cf app myapp --guid
				GUID: app.Guid,
				// app must be start
				State: app.State,
			},
		},
		// this is a signature for ssh, it can be found when doing cf curl /v2/info
		sshInfo.AppSSHHostKeyFingerprint,
		// endpoint to connect in ssh, it can be found when doing cf curl /v2/info
		sshInfo.AppSSHEndpoint,
		// token retrieve when doing cf ssh-code
		token,
	)
	err = secureShell.Connect(&options.SSHOptions{
		AppName: appName,
		SkipHostValidation: sslDisabled,
		SkipRemoteExecution: true,
		Command: []string{},
		Index: uint(0),
		ForwardSpecs: []options.ForwardSpec{},
		TerminalRequest: options.RequestTTYAuto,
	})
	if err != nil {
		return err
	}
	defer secureShell.Close()
	logger.Info("Finished authenticating for ssh.")
	keepaliveStopCh := make(chan struct{})
	defer close(keepaliveStopCh)

	go keepalive(secureShell.secureClient.Conn(), time.NewTicker(keepAliveInterval), keepaliveStopCh)

	containerFiler := NewContainerFiler(secureShell.secureClient)
	containerFiler.OutWriter = os.Stdout
	sync, err := NewSync(containerFiler, sourceDir, targetDir)
	if err != nil {
		return err
	}
	return sync.Run()
}