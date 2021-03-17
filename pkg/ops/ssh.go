package ops

import (
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"net"
	"os"
	"os/user"
)

// SshProgClient is an ssh client designed to do remote commands or RPC's
type SshProgClient struct {
	Host   string
	Port   int
	Config *ssh.ClientConfig
}

// NewSshProgClient creates a client for the given host, port, and config.
func NewSshProgClient(host string, port int, config *ssh.ClientConfig) (client *SshProgClient) {
	client = &SshProgClient{
		Host:   host,
		Port:   port,
		Config: config,
	}

	return client
}

// SshClient generates an SSH client for talking to the provisioning server
func SshClient(hostname string, port int, username string) (client *SshProgClient, err error) {
	var operator string
	if username == "" {
		userobj, err := user.Current()
		if err != nil {
			err = errors.Wrapf(err, "failed to determine local user")
			return client, err
		}

		operator = userobj.Username
	} else {
		operator = username
	}

	sshConfig := &ssh.ClientConfig{
		User: operator,
		Auth: []ssh.AuthMethod{
			SSHAgent(),
		},
		//HostKeyCallback: ssh.FixedHostKey(HostKey()),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client = NewSshProgClient(hostname, port, sshConfig)

	return client, err
}

// SSHAgent is a programmatic client that talks to the ssh agent.
func SSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

// SCPFile copies a file via SCP to the remote host.
func (c *SshProgClient) SCPFile(content string, filename string) (err error) {

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)

	connection, err := ssh.Dial("tcp", addr, c.Config)
	if err != nil {
		err = errors.Wrapf(err, "failed to dial server")
		return err
	}

	session, err := connection.NewSession()
	if err != nil {
		err = errors.Wrapf(err, "failed to create connection")
		return err
	}

	go func() {
		w, _ := session.StdinPipe()

		// this has to have a new line
		_, _ = fmt.Fprintln(w, "C0644", len(content), filename)

		// this is whatever the content is
		_, _ = fmt.Fprint(w, content)

		// this cannot have a new line
		_, _ = fmt.Fprint(w, "\x00")

		_ = w.Close()
	}()

	/*
			scp transfers by opening an ssh connection and opening another copy of scp on the remote system.  One instances sends, the other receives and copies to it's local disk.

		 -t tells scp that it was invoked by another instance and it will be receiving.

		 -f tells that it was involed by another instance and it should send.

		 These options are totally undocumented.

	*/

	err = session.Run("/usr/bin/scp -tr ./")
	if err != nil {
		err = errors.Wrapf(err, "Failed to copy file")
		return err
	}

	_ = session.Close()

	return err
}

// RpcCall flings bytes at a remote server over SSH to STDIN, and receives whatever that
// server decides to send back on STDOUT and STDERR.  What you send it, and what you do with the
// reply is between you and the server.
func (c *SshProgClient) RpcCall(input []byte, stdout, stderr io.Writer) (err error) {
	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)

	connection, err := ssh.Dial("tcp", addr, c.Config)
	if err != nil {
		err = errors.Wrapf(err, "failed to dial server")
		return err
	}

	session, err := connection.NewSession()
	if err != nil {
		err = errors.Wrapf(err, "failed to create connection")
		return err
	}

	session.Stdout = stdout
	session.Stderr = stderr

	err = session.Start(string(input))
	if err != nil {
		err = errors.Wrapf(err, "failed to open shell on remote server")
		return err
	}

	err = session.Wait()
	if err != nil {
		err = errors.Wrapf(err, "error waiting for session to complete")
		return err
	}

	// It probably closes serverside before this is necessary, but let's be thorough.
	_ = session.Close()

	return err
}
