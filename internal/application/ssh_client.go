package application

type SSHClient interface {
	// RunCommand executes a command on the remote server and streams output to logChan.
	RunCommand(ip string, port int, cmd string, logChan chan<- string) error
}
