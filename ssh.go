package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/ssh/options"
)

const (
	md5FingerprintLength = 47 // inclusive of space between bytes
	hexSha1FingerprintLength = 59 // inclusive of space between bytes
	base64Sha256FingerprintLength = 43
)


//go:generate counterfeiter . SecureDialer

type SecureDialer interface {
	Dial(network, address string, config *ssh.ClientConfig) (*SecureClient, error)
}


//go:generate counterfeiter . ListenerFactory

type ListenerFactory interface {
	Listen(network, address string) (net.Listener, error)
}

func NewSecureShell(
secureDialer SecureDialer,
listenerFactory ListenerFactory,
keepAliveInterval time.Duration,
app models.Application,
sshEndpointFingerprint string,
sshEndpoint string,
token string,
) *SecureShell {
	return &SecureShell{
		secureDialer:      secureDialer,
		listenerFactory:   listenerFactory,
		keepAliveInterval: keepAliveInterval,
		app:               app,
		sshEndpointFingerprint: sshEndpointFingerprint,
		sshEndpoint:            sshEndpoint,
		token:                  token,
		localListeners:         []net.Listener{},
	}
}

type SecureShell struct {
	secureDialer           SecureDialer
	listenerFactory        ListenerFactory
	keepAliveInterval      time.Duration
	app                    models.Application
	sshEndpointFingerprint string
	sshEndpoint            string
	token                  string
	secureClient           *SecureClient
	opts                   *options.SSHOptions

	localListeners         []net.Listener
}

func (c *SecureShell) Connect(opts *options.SSHOptions) error {
	err := c.validateTarget(opts)
	if err != nil {
		return err
	}

	clientConfig := &ssh.ClientConfig{
		User: fmt.Sprintf("cf:%s/%d", c.app.GUID, opts.Index),
		Auth: []ssh.AuthMethod{
			ssh.Password(c.token),
		},
		HostKeyCallback: fingerprintCallback(opts, c.sshEndpointFingerprint),
	}

	secureClient, err := c.secureDialer.Dial("tcp", c.sshEndpoint, clientConfig)
	if err != nil {
		return err
	}

	c.secureClient = secureClient
	c.opts = opts
	return nil
}

func (c *SecureShell) Close() error {
	for _, listener := range c.localListeners {
		_ = listener.Close()
	}
	return c.secureClient.Close()
}

func (c *SecureShell) LocalPortForward() error {
	for _, forwardSpec := range c.opts.ForwardSpecs {
		listener, err := c.listenerFactory.Listen("tcp", forwardSpec.ListenAddress)
		if err != nil {
			return err
		}
		c.localListeners = append(c.localListeners, listener)

		go c.localForwardAcceptLoop(listener, forwardSpec.ConnectAddress)
	}

	return nil
}

func (c *SecureShell) localForwardAcceptLoop(listener net.Listener, addr string) {
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return
		}

		go c.handleForwardConnection(conn, addr)
	}
}

func (c *SecureShell) handleForwardConnection(conn net.Conn, targetAddr string) {
	defer conn.Close()

	target, err := c.secureClient.Dial("tcp", targetAddr)
	if err != nil {
		fmt.Printf("connect to %s failed: %s\n", targetAddr, err.Error())
		return
	}
	defer target.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go copyAndClose(wg, conn, target)
	go copyAndClose(wg, target, conn)
	wg.Wait()
}

func copyAndClose(wg *sync.WaitGroup, dest io.WriteCloser, src io.Reader) {
	_, _ = io.Copy(dest, src)
	_ = dest.Close()
	if wg != nil {
		wg.Done()
	}
}

func copyAndDone(wg *sync.WaitGroup, dest io.Writer, src io.Reader) {
	_, _ = io.Copy(dest, src)
	wg.Done()
}

func (c *SecureShell) Wait() error {
	keepaliveStopCh := make(chan struct{})
	defer close(keepaliveStopCh)

	go keepalive(c.secureClient.Conn(), time.NewTicker(c.keepAliveInterval), keepaliveStopCh)

	return c.secureClient.Wait()
}

func (c *SecureShell) validateTarget(opts *options.SSHOptions) error {
	if strings.ToUpper(c.app.State) != "STARTED" {
		return fmt.Errorf("Application %q is not in the STARTED state", opts.AppName)
	}

	if !c.app.Diego {
		return fmt.Errorf("Application %q is not running on Diego", opts.AppName)
	}

	return nil
}

func md5Fingerprint(key ssh.PublicKey) string {
	sum := md5.Sum(key.Marshal())
	return strings.Replace(fmt.Sprintf("% x", sum), " ", ":", -1)
}

func hexSha1Fingerprint(key ssh.PublicKey) string {
	sum := sha1.Sum(key.Marshal())
	return strings.Replace(fmt.Sprintf("% x", sum), " ", ":", -1)
}

func base64Sha256Fingerprint(key ssh.PublicKey) string {
	sum := sha256.Sum256(key.Marshal())
	return base64.RawStdEncoding.EncodeToString(sum[:])
}

type hostKeyCallback func(hostname string, remote net.Addr, key ssh.PublicKey) error

func fingerprintCallback(opts *options.SSHOptions, expectedFingerprint string) hostKeyCallback {
	if opts.SkipHostValidation {
		return nil
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		var fingerprint string

		switch len(expectedFingerprint) {
		case base64Sha256FingerprintLength:
			fingerprint = base64Sha256Fingerprint(key)
		case hexSha1FingerprintLength:
			fingerprint = hexSha1Fingerprint(key)
		case md5FingerprintLength:
			fingerprint = md5Fingerprint(key)
		case 0:
			fingerprint = md5Fingerprint(key)
			return fmt.Errorf("Unable to verify identity of host.\n\nThe fingerprint of the received key was %q.", fingerprint)
		default:
			return errors.New("Unsupported host key fingerprint format")
		}

		if fingerprint != expectedFingerprint {
			return fmt.Errorf("Host key verification failed.\n\nThe fingerprint of the received key was %q.", fingerprint)
		}
		return nil
	}
}

func (c *SecureShell) shouldAllocateTerminal(opts *options.SSHOptions, stdinIsTerminal bool) bool {
	switch opts.TerminalRequest {
	case options.RequestTTYForce:
		return true
	case options.RequestTTYNo:
		return false
	case options.RequestTTYYes:
		return stdinIsTerminal
	case options.RequestTTYAuto:
		return len(opts.Command) == 0 && stdinIsTerminal
	default:
		return false
	}
}

func keepalive(conn ssh.Conn, ticker *time.Ticker, stopCh chan struct{}) {
	for {
		select {
		case <-ticker.C:
			_, _, _ = conn.SendRequest("keepalive@cloudfoundry.org", true, nil)
		case <-stopCh:
			ticker.Stop()
			return
		}
	}
}

func (c *SecureShell) terminalType() string {
	term := os.Getenv("TERM")
	if term == "" {
		term = "xterm"
	}
	return term
}

type secureDialer struct{}

func (d *secureDialer) Dial(network string, address string, config *ssh.ClientConfig) (*SecureClient, error) {
	client, err := ssh.Dial(network, address, config)
	if err != nil {
		return nil, err
	}

	return &SecureClient{client: client}, nil
}

func DefaultSecureDialer() SecureDialer {
	return &secureDialer{}
}

type SecureClient struct{ client *ssh.Client }

func (sc *SecureClient) Close() error {
	return sc.client.Close()
}
func (sc *SecureClient) Conn() ssh.Conn {
	return sc.client.Conn
}
func (sc *SecureClient) Wait() error {
	return sc.client.Wait()
}
func (sc *SecureClient) Dial(n, addr string) (net.Conn, error) {
	return sc.client.Dial(n, addr)
}
func (sc *SecureClient) NewSession() (*ssh.Session, error) {
	return sc.client.NewSession()
}

type listenerFactory struct{}

func (lf *listenerFactory) Listen(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

func DefaultListenerFactory() ListenerFactory {
	return &listenerFactory{}
}
