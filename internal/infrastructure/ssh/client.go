package ssh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	config *ssh.ClientConfig
}

func NewClient() (*Client, error) {
	// In a real production system, you'd manage keys securely or per-project.
	// Here, for agility, we attempt to use the local user's ~/.ssh/id_rsa.
	
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	
	keyPath := filepath.Join(home, ".ssh", "id_rsa")
	key, err := os.ReadFile(keyPath)
	if err != nil {
		// Fallback to password or empty config if no key found (for demo purposes)
		// Or we can return error and force user to have an id_rsa
		// return nil, fmt.Errorf("unable to read private key: %v", err)
	}

	var authMethods []ssh.AuthMethod
	if key != nil {
		signer, err := ssh.ParsePrivateKey(key)
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}
	
	// A fallback password could be loaded from env, or server specific config
	// authMethods = append(authMethods, ssh.Password("password"))

	config := &ssh.ClientConfig{
		User: "root", // Hardcoded to root for testing, should be configurable
		Auth: authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Do not do this in strict prod
	}

	return &Client{config: config}, nil
}

func (c *Client) RunCommand(ip string, port int, cmd string, logChan chan<- string) error {
	addr := fmt.Sprintf("%s:%d", ip, port)
	client, err := ssh.Dial("tcp", addr, c.config)
	if err != nil {
		logChan <- fmt.Sprintf("[SSH] Failed to dial %s: %v\n", addr, err)
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		logChan <- fmt.Sprintf("[SSH] Failed to create session: %v\n", err)
		return err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	go streamOutput(stdout, logChan)
	go streamOutput(stderr, logChan)

	logChan <- fmt.Sprintf("[SSH] Executing on %s: %s\n", ip, cmd)
	err = session.Run(cmd)
	if err != nil {
		logChan <- fmt.Sprintf("[SSH] Command failed: %v\n", err)
		return err
	}

	logChan <- fmt.Sprintf("[SSH] Command successful on %s\n", ip)
	return nil
}

func streamOutput(r io.Reader, logChan chan<- string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		logChan <- scanner.Text() + "\n"
	}
}
