package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"

	"github.com/pkg/errors"

	"gopkg.in/inconshreveable/log15.v2"

	"golang.org/x/net/context"

	"strings"

	"bufio"
)

var (
	// Application is the application name to set to tag logs
	Application string

	// Binary is the executable to wrap
	Binary string

	// Args is the list of argument to pass to the main executable
	Args string

	// Prefix an optional prefix for each log line
	Prefix string

	// Loghost is the host:port to which the logs should be sent
	Loghost string

	log log15.Logger
)

func init() {
	flag.StringVar(&Application, "application", "", "Application log under")
	flag.StringVar(&Binary, "binary", "", "Binary to invoke")
	flag.StringVar(&Args, "arguments", "", "Arguments to pass to binary")
	flag.StringVar(&Prefix, "prefix", "", "Prefix each log line with this value")
	flag.StringVar(&Loghost, "loghost", "", "Logging host:port")
}

func main() {
	var retcode int

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	log = log15.New()
	if Application != "" {
		log = log.New("application", Application)
	}

	err := setDestination(Loghost)
	if err != nil {
		log.Crit("failed to set log destination", "error", err)
		os.Exit(1)
	}

	// Wait for OS to signal stop
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)

		// wait for unix signal
		select {
		case <-ctx.Done():
			return
		case s := <-c:
			log.Warn("Got signal; stopping logshipper", "signal", s)
			cancel()
		}
	}()

	// Start the downstream command
	args := []string{}
	if Args != "" {
		for _, a := range strings.Split(Args, " ") {
			args = append(args, a)
		}
	}
	cmd := exec.Command(Binary, args...)
	go func() {
		err := Exec(cmd)
		if err != nil {
			fmt.Println("command failed:", err.Error())
			retcode = 1
		}
		cancel()
	}()

	<-ctx.Done()

	// Make sure the downstream process is dead
	log.Debug("Killing process")
	if cmd.Process != nil {
		cmd.Process.Kill()
	}

	os.Exit(retcode)
}

// Exec initializes the state and launches the
// logshipping subprocess
func Exec(cmd *exec.Cmd) error {
	log.Debug("Executing command", "command", Binary, "arguments", Args)

	// Get subprocess pipe
	stderr, err := cmd.StderrPipe()

	if err != nil {
		log.Error("Failed to pipe stderr", "command", Binary, "error", err)
		return err
	}

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		log.Error("Failed to pipe stdout", "command", Binary, "error", err)
		return err
	}

	// Start command in background
	err = cmd.Start()

	if err != nil {
		log.Error("Failed to pipe stdin", "command", Binary, "error", err)
		return err
	}

	go func() {
		r := bufio.NewReader(stderr)
		var err error
		for err == nil {
			var line []byte
			line, _, err = r.ReadLine()
			if err == nil {
				writeLogEntry(Prefix, "stderr", string(line))
			}
		}
	}()

	go func() {
		r := bufio.NewReader(stdout)
		var err error
		for err == nil {
			var line []byte
			line, _, err = r.ReadLine()
			if err == nil {
				writeLogEntry(Prefix, "stdout", string(line))
			}
		}
	}()

	return cmd.Wait()
}

func writeLogEntry(outputPrefix string, channel string, line string) {
	if strings.Contains(line, "ERROR") {
		log.Error(outputPrefix+" "+line, "channel", channel)
	} else if strings.Contains(line, "DEBUG") {
		log.Debug(outputPrefix+" "+line, "channel", channel)
	} else if strings.Contains(line, "WARN") {
		log.Warn(outputPrefix+" "+line, "channel", channel)
	} else {
		log.Info(outputPrefix+" "+line, "channel", channel)
	}
}

func setDestination(addr string) error {
	var err error

	h := log15.StdoutHandler
	if addr != "" {
		h, err = log15.NetHandler("udp", addr, log15.JsonFormat())
		if err != nil {
			return errors.Wrap(err, "failed to set log destination")
		}
	}
	log.SetHandler(h)
	return nil
}
