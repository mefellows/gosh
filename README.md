# GoSH - Go PowerShell
Cross platform, interactive remote PowerShell runner, written in Go.

GoSH is Go gettable: `go get github.com/mefellows/gosh`

[![wercker status](https://app.wercker.com/status/da70aaf46f83e3af87471b119c3ba13d/s "wercker status")](https://app.wercker.com/project/bykey/da70aaf46f83e3af87471b119c3ba13d)

## Usage

1. [Download](releases) the latest release, and put `gosh` somewhere on the PATH.
1. Run GoSH

	```
    mfellows ~/tmp $ gosh --help
    Usage of gosh:
      -debug=false: Output debugging info
      -host="localhost": Remote host
      -password="": Remote admin password
      -port=5985: Remote port
      -timeout="PT36000S": Remote timeout
      -username="vagrant": Remote admin username
    ```

1. Typically, this would look something like:

	```
	mfellows ~/tmp $ gosh --host foo.com --username matt
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
    
    Enter Password:    
    **********

    > whoami
    
    > mfapi\vagrant
    ```

1.

## Features

* Remote Windows shell (PowerShell) invocation from Linux, MacOSX and Windows
* "sudo" support (runs as an elevated process, albeit a bit slower)
* Not Windows

## Setup

GoSH uses WinRM for communication with the remote machine, and requires it to be configured before it can work:

```
winrm quickconfig -q
```

For a more comprehensive setup, consider:

```
cmd.exe /c winrm quickconfig -q
cmd.exe /c winrm quickconfig '-transport:http'
cmd.exe /c winrm set "winrm/config" '@{MaxTimeoutms="1800000"}'
cmd.exe /c winrm set "winrm/config/winrs" '@{MaxMemoryPerShellMB="1024"}'
cmd.exe /c winrm set "winrm/config/service" '@{AllowUnencrypted="true"}'
cmd.exe /c winrm set "winrm/config/client" '@{AllowUnencrypted="true"}'
cmd.exe /c winrm set "winrm/config/service/auth" '@{Basic="true"}'
cmd.exe /c winrm set "winrm/config/client/auth" '@{Basic="true"}'
cmd.exe /c winrm set "winrm/config/service/auth" '@{CredSSP="true"}'
cmd.exe /c winrm set "winrm/config/listener?Address=*+Transport=HTTP" '@{Port="5985"}'
cmd.exe /c netsh advfirewall firewall set rule group="remote administration" new enable=yes
cmd.exe /c netsh firewall add portopening TCP 5985 "Port 5985"
cmd.exe /c net stop winrm
cmd.exe /c sc config winrm start= auto
cmd.exe /c net start winrm
```


## TODO

* Implementation of a more elegant parser, lexer and grammar
* Support Secure WinRM communications via certificates
* History
* ctrl-r lookup support
* Basic variable interpolation
* [Winrmcp](https://github.com/packer-community/winrmcp) support for simple remote copy/paste 
* Considering creating a daemon on the Windows host to avoid need for WinRM altogether
