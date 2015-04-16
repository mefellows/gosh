package commands

var commands = []string{
	"reconnect",
	"quit",
	"sudo",
}

func GetGoshCommands() []string {
	return commands
}
