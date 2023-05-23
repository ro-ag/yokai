package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	zmq "github.com/go-zeromq/zmq4"
)

type Daemon struct {
	cmd  *exec.Cmd
	sock zmq.Socket
}

func NewDaemon() *Daemon {
	cmd := exec.Command("python3", "/path/to/daemon.py")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		fmt.Println("Failed to start daemon:", err)
		os.Exit(1)
	}

	sock := zmq.NewReq(nil)

	err = sock.Dial("ipc:///tmp/daemon.ipc") // IPC socket
	if err != nil {
		fmt.Println("Failed to dial socket:", err)
		os.Exit(1)
	}

	return &Daemon{
		cmd:  cmd,
		sock: sock,
	}
}

func (d *Daemon) ExecuteScript(scriptPath string) (string, error) {
	msg := zmq.NewMsgString(scriptPath)
	err := d.sock.Send(msg)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %v", err)
	}

	reply, err := d.sock.Recv()
	if err != nil {
		return "", fmt.Errorf("failed to receive message: %v", err)
	}

	return string(reply.Bytes()), nil
}

func (d *Daemon) MonitorDaemon(ctx context.Context) {
	// Goroutine to wait for daemon to exit
	done := make(chan error, 1)
	go func() {
		done <- d.cmd.Wait()
	}()

	// Monitor daemon
	for {
		select {
		case err := <-done:
			if err != nil {
				fmt.Println("Daemon exited with error:", err)
			} else {
				fmt.Println("Daemon exited normally")
			}
			// Restart daemon if it stops
			d.cmd.Start()
			go func() {
				done <- d.cmd.Wait()
			}()
		case <-ctx.Done():
			return
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	daemon := NewDaemon()

	go daemon.MonitorDaemon(ctx)

	result, err := daemon.ExecuteScript("/path/to/your/script.py --arg1 --arg2")
	if err != nil {
		fmt.Println("Failed to execute script:", err)
	} else {
		fmt.Println("Received:", result)
	}

	// Call cancel when you want to stop the daemon monitoring
	//cancel()

	// To prevent main from exiting
	select {}
}
