package main

import (
	"fmt"
	"github.com/masterzen/winrm/winrm"
	"github.com/mefellows/gosh/commands"
	"github.com/mitchellh/cli"
	"github.com/peterh/liner"
	"log"
	"os"
	"runtime"
	"strings"
)

type Request struct {
	command  string
	elevated bool
}
type Response struct {
	exitCode int
	err      error
	stdOut   string
	stdErr   string
}

type GoShell struct {
	buffer []byte
	client *winrm.Client
	config *ConnectionConfig
	ui     cli.Ui
}

type ConnectionConfig struct {
	Hostname string
	Port     int
	Username string
	Password string
	Timeout  string
}

var historyFile = "/tmp/.liner_history"
var completions = append(commands.GetPowershellCommands(), append(commands.GetGoshCommands(), commands.GetPowershellOperators()...)...)

func NewShell(config *ConnectionConfig) (*GoShell, error) {
	client, err := winrm.NewClientWithParameters(&winrm.Endpoint{Host: config.Hostname, Port: config.Port, HTTPS: false, Insecure: true, CACert: nil}, config.Username, config.Password, winrm.NewParameters(config.Timeout, "en-US", 153600))

	if err != nil {
		return nil, err
	}
	var ui cli.Ui

	if runtime.GOOS == "windows" {
		ui = &cli.BasicUi{Writer: os.Stdout, Reader: os.Stdin, ErrorWriter: os.Stderr}
	} else {
		ui = &cli.ColoredUi{
			Ui:          &cli.BasicUi{Writer: os.Stdout, Reader: os.Stdin, ErrorWriter: os.Stderr},
			OutputColor: cli.UiColorYellow,
			InfoColor:   cli.UiColorNone,
			ErrorColor:  cli.UiColorRed,
		}
	}

	return &GoShell{
		buffer: make([]byte, 0),
		config: config,
		client: client,
		ui:     ui,
	}, nil
}

func setupLiner() *liner.State {
	line := liner.NewLiner()

	if f, err := os.Open(historyFile); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	// Autocompletes common commands/args. Not perfect, but handy
	line.SetCompleter(func(line string) (c []string) {

		// If it's a command with args / spaces, we need to tokenize the input
		// So we just complete the last incomplete statement
		toComplete := line
		prefix := ""
		i := strings.LastIndex(line, " ")
		if i > 0 {
			toComplete = strings.TrimSpace(line[i:])
			prefix = line[:i+1]
		}

		for _, n := range completions {
			if strings.HasPrefix(strings.ToLower(n), strings.ToLower(toComplete)) {
				c = append(c, fmt.Sprintf("%s%s", prefix, n))
			}
		}
		return
	})

	return line
}

func (s *GoShell) readInput() (string, error) {
	liner := setupLiner()
	liner.SetCtrlCAborts(true)
	defer liner.Close()

	input, err := liner.Prompt("$ ")
	if err != nil {
		log.Print("Error reading line: ", err)
	} else {
		liner.AppendHistory(input)

		// Save to file, but do it asynchronously
		go func() {
			if f, err := os.Create(historyFile); err != nil {
				log.Print("Error writing history file: ", err)
			} else {
				liner.WriteHistory(f)
				f.Close()
			}
		}()
	}
	liner.Close()
	return input, err
}

// Create a prompt and read from it
func (s *GoShell) waitForInput(fp *os.File, writeChan chan string, quitChan chan<- bool) {
	go func() {
		for {
			line, err := s.readInput()

			switch {
			case strings.TrimSpace(line) == "":
				break
			}
			if err != nil || line == "exit" || line == "quit" {
				if err != nil {
					fmt.Printf("Error: %s", err.Error())
				}
				quitChan <- true
				return
			}

			writeChan <- line
			return
		}
	}()
}

func (s *GoShell) shell(fp *os.File) {
	// main shell loop
	if fp == nil {
		fp = os.Stdin
	}

	// quit channel
	quitChan := make(chan bool)
	// inputChan channel
	inputChan := make(chan string)

loop:
	for {
		s.waitForInput(fp, inputChan, quitChan)

		select {
		case i := <-inputChan:
			r := &Request{command: i, elevated: false}

			// Parse input grammer
			switch {
			case strings.Index(i, "sudo") == 0:
				r.command = strings.SplitAfter(r.command, "sudo")[1]
				r.elevated = true
			}
			posh := &Powershell{s.client}
			res := posh.runCommand(r)
			if res.stdOut != "" {
				s.ui.Output(res.stdOut)
			}
			if res.stdErr != "" {
				s.ui.Error(res.stdErr)
			}

			if res.err != nil {
				s.ui.Error(res.err.Error())
			}
			fmt.Println()
		case <-quitChan:
			s.ui.Info("Quitting...")
			break loop
		}
	}

	return
}

func (s *GoShell) Close() {
	//s.input.Close()
	return
}
