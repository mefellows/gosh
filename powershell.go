package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/masterzen/winrm/winrm"
	"github.com/mitchellh/packer/common/uuid"
	"github.com/packer-community/winrmcp/winrmcp"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

type elevatedShellOptions struct {
	Command  string
	User     string
	Password string
}

type Powershell struct {
	client *winrm.Client
}

func (p *Powershell) runCommand(request *Request) *Response {
	response := &Response{exitCode: 0}

	var exitCode int
	if !request.elevated {
		log.Printf("Running remote command...")
		response.stdOut, response.stdErr, response.exitCode, response.err = p.client.RunWithString(winrm.Powershell(request.command), "")
		response.exitCode = exitCode
	} else {
		log.Printf("Running elevated remote command...")
		// TODO: Capture response fields
		response.err = p.StartElevated(request.command)
	}

	return response
}

func (p *Powershell) StartElevated(cmd string) (err error) {
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
	_, err = p.client.RunWithInput(command, os.Stdout, os.Stderr, os.Stdin)
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
