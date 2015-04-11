package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/masterzen/winrm/winrm"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/packer/common/uuid"
	"github.com/packer-community/winrmcp/winrmcp"
	"github.com/peterh/liner"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

type Request struct {
	command  string
	elevated bool
}
type Response struct {
	exitCode int
	response string
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
	historyFile := "/tmp/.liner_history"

	if f, err := os.Open(); err == nil {
		line.ReadHistory(f)
		f.Close()
	}
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
			if f, err := os.Create(history_fn); err != nil {
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

			fmt.Printf("Line, about to send: %s", line)
			writeChan <- line
			fmt.Printf("sent")
			return
		}
	}()
}

func (s *GoShell) runCommand(request *Request) *Response {
	var err error
	response := &Response{exitCode: 0}

	var stdOut, errOut string
	var exitCode int
	if !request.elevated {
		log.Printf("Running remote command...")
		stdOut, errOut, exitCode, err = s.client.RunWithString(winrm.Powershell(request.command), "")
		response.exitCode = exitCode
		if stdOut != "" {
			s.ui.Output(stdOut)
		}
		if errOut != "" {
			s.ui.Error(errOut)
		}
	} else {
		log.Printf("Running elevated remote command...")
		err = s.StartElevated(request.command)
	}

	if err != nil {
		s.ui.Error(err.Error())
		return response
	}

	return response
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
			s.runCommand(r)
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

type elevatedShellOptions struct {
	Command  string
	User     string
	Password string
}

func (s *GoShell) StartElevated(cmd string) (err error) {
	// The command gets put into an interpolated string in the PS script,
	// so we need to escape any embedded quotes.
	cmd = strings.Replace(cmd, "\"", "`\"", -1)

	elevatedScript, err := createCommandText(cmd)

	if err != nil {
		return err
	}

	// Upload the script which creates and manages the scheduled task
	winrmcp, err := winrmcp.New(fmt.Sprintf("%s:%d", host, port), &winrmcp.Config{
		Auth:                  winrmcp.Auth{user, pass},
		OperationTimeout:      time.Second * 60,
		MaxOperationsPerShell: 15,
	})
	tmpFile, err := ioutil.TempFile(os.TempDir(), "gosh-elevated-shell.ps1")
	log.Printf("Temp file: %s", tmpFile.Name())

	writer := bufio.NewWriter(tmpFile)
	if _, err := writer.WriteString(elevatedScript); err != nil {
		return fmt.Errorf("Error preparing shell script: %s", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("Error preparing shell script: %s", err)
	}

	tmpFile.Close()

	err = winrmcp.Copy(tmpFile.Name(), "${env:TEMP}/gosh-elevated-shell.ps1")

	if err != nil {
		log.Printf("Error copying shell script: %s", err)
		return err
	}

	// Run the script that was uploaded
	command := fmt.Sprintf("powershell -executionpolicy bypass -file \"%s\"", "%TEMP%\\gosh-elevated-shell.ps1")
	log.Printf("Running script: %s", command)
	_, err = s.client.RunWithInput(command, os.Stdout, os.Stderr, os.Stdin)
	return err
}

func createCommandText(cmd string) (command string, err error) {

	log.Printf("Building elevated command for: %s", cmd)

	// generate command
	var buffer bytes.Buffer
	err = elevatedTemplate.Execute(&buffer, elevatedOptions{
		User:            user,
		Password:        pass,
		TaskDescription: "GoSH elevated task",
		TaskName:        fmt.Sprintf("gosh-%s", uuid.TimeOrderedUUID()),
		EncodedCommand:  powershellEncode([]byte(cmd + "; exit $LASTEXITCODE")),
	})

	if err != nil {
		return "", err
	}

	log.Printf("ELEVATED SCRIPT: %s\n\n", string(buffer.Bytes()))
	return string(buffer.Bytes()), nil

}

func powershellEncode(buffer []byte) string {
	// 2 byte chars to make PowerShell happy
	wideCmd := ""
	for _, b := range buffer {
		wideCmd += string(b) + "\x00"
	}

	// Base64 encode the command
	input := []uint8(wideCmd)
	return base64.StdEncoding.EncodeToString(input)
}
