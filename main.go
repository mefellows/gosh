package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

var (
	host     string
	user     string
	pass     string
	port     int
	elevated bool
	debug    bool
	timeout  string
)

func main() {

	flag.StringVar(&host, "host", "localhost", "winrm host")
	flag.StringVar(&user, "username", "vagrant", "winrm admin username")
	flag.StringVar(&pass, "password", "vagrant", "winrm admin password")
	flag.StringVar(&timeout, "timeout", "PT36000S", "winrm timeout")
	flag.IntVar(&port, "port", 5985, "winrm port")
	flag.BoolVar(&debug, "debug", false, "output debugging info")
	flag.Parse()

	if !debug {
		log.SetOutput(ioutil.Discard)
	}

	fmt.Printf(`
Welcome to

 .d8888b.            .d8888b.   888               888 888 TM
 d88P  Y88b          d88P  Y88b 888               888 888 
 888    888          Y88b.      888               888 888 
 888         .d88b.   "Y888b.   88888b.   .d88b.  888 888 
 888  88888 d88""88b     "Y88b. 888 "88b d8P  Y8b 888 888 
 888    888 888  888       "888 888  888 88888888 888 888 
 Y88b  d88P Y88..88P Y88b  d88P 888  888 Y8b.     888 888 
  "Y8888P88  "Y88P"   "Y8888P"  888  888  "Y8888  888 888 
                                                           
 'Coz even Remote-PSSession sucks!  

`)

	config := &ConnectionConfig{
		Hostname: host,
		Username: user,
		Password: pass,
		Port:     port,
		Timeout:  timeout,
	}

	shell, err := NewShell(config)
	if err != nil {
		fmt.Printf("Unable to start shell: %s", err.Error())
		os.Exit(1)
	}

	// Start shell!
	shell.shell(nil)
	os.Exit(0)
}
