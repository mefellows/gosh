package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

type Request struct{}
type Response struct{}
type History []map[Request]Response
type Shell interface{}

type GoShell struct {
	buffer  []byte
	history History
}

type ConnectionConfig struct {
	Hostname string
	port     int
	username string
	password string
}

func NewShell() *GoShell {
	return &GoShell{
		buffer:  make([]byte, 0),
		history: make([]map[Request]Response, 0),
	}
}

func (s *GoShell) waitForInput(fp *os.File, writeChan chan<- string, quitChan chan<- bool) {
	// read from shell prompt
	go func() {
		reader := bufio.NewReader(fp)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				quitChan <- true
				return
			}

			// append token.SEMICOLON
			line = append(line, 59)

			// talk to writeChan chan?
			writeChan <- string(line)
		}
	}()
}

func (s *GoShell) runCommand(input string) {
	log.Printf("Received input: %s", input)

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
			s.runCommand(i)
		case <-quitChan:
			fmt.Println("[GoSH] terminated")
			break loop
		}
	}

	return
}

func main() {
	fmt.Printf("Welcome to GoSH - start shelling! \n")
	shell := NewShell()
	shell.shell(nil)
}
