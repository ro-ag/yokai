package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	zmq "github.com/go-zeromq/zmq4"
)

type Daemon struct {
	cmd  *exec.Cmd
	sock zmq.Socket
}

func NewDaemon() (*Daemon, error) {
	cmd := exec.Command("python3", "./daemon/executor.py")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start daemon: %v", err)
	}

	// Goroutine to check for errors from the Python script
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Wait()
	}()

	// Check for errors after a short delay
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("daemon encountered an error: %v", err)
		}
	case <-time.After(time.Second):
		// No error from the daemon after 1 second, so proceed
	}

	sock := zmq.NewReq(nil)

	err = sock.Dial("ipc:///tmp/daemon.ipc") // IPC socket
	if err != nil {
		return nil, fmt.Errorf("failed to dial socket: %v", err)
	}

	return &Daemon{
		cmd:  cmd,
		sock: sock,
	}, nil
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

	daemon, err := NewDaemon()
	if err != nil {
		log.Fatalf("error to start daemon: %v", err)
	}

	go daemon.MonitorDaemon(ctx)

	for i := 0; i < 100; i++ {
		result, err := daemon.ExecuteScript("./example.py")
		if err != nil {
			fmt.Println("Failed to execute script:", err)
		} else {
			fmt.Println("Received:", result)
		}
	}

	fmt.Println("shutting down")

	// Call cancel when you want to stop the daemon monitoring
	cancel()

	// To prevent main from exiting
	//select {}
}
