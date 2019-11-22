// Package cli is a command line interface
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/micro/cli"
)

var (
	prompt = "micro> "

	commands = map[string]*command{
		"quit":    &command{"quit", "Exit the CLI", quit},
		"exit":    &command{"exit", "Exit the CLI", quit},
		"call":    &command{"call", "Call a service", callService},
		"list":    &command{"list", "List services, peers or routes", list},
		"stream":  &command{"stream", "Stream a call to a service", streamService},
		"publish": &command{"publish", "Publish a message to a topic", publish},
		"health":  &command{"health", "Get service health", queryHealth},
	}
)

type command struct {
	name  string
	usage string
	exec  exec
}

func runc(c *cli.Context) {
	commands["help"] = &command{"help", "CLI usage", help}
	alias := map[string]string{
		"?":  "help",
		"ls": "list",
	}

	r, err := readline.New(prompt)
	if err != nil {
		fmt.Fprint(os.Stdout, err)
		os.Exit(1)
	}
	defer r.Close()

	for {
		args, err := r.Readline()
		if err != nil {
			fmt.Fprint(os.Stdout, err)
			return
		}

		args = strings.TrimSpace(args)

		// skip no args
		if len(args) == 0 {
			continue
		}

		parts := strings.Split(args, " ")
		if len(parts) == 0 {
			continue
		}

		name := parts[0]

		// get alias
		if n, ok := alias[name]; ok {
			name = n
		}

		if cmd, ok := commands[name]; ok {
			rsp, err := cmd.exec(c, parts[1:])
			if err != nil {
				println(err.Error())
				continue
			}
			println(string(rsp))
		} else {
			println("unknown command")
		}
	}
}

//HealthCommands Query the health of a service
func HealthCommands() []cli.Command {
	return []cli.Command{
		{
			Name:   "check",
			Usage:  "Query the health of a service",
			Action: printer(queryHealth),
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "address",
					Usage:  "Set the address of the service instance to call",
					EnvVar: "MICRO_ADDRESS",
				},
			},
		},
	}
}

//Commands Run the interactive CLI
func Commands() []cli.Command {
	commands := []cli.Command{
		{
			Name:   "cli",
			Usage:  "Run the interactive CLI",
			Action: runc,
		},
		{
			Name:   "call",
			Usage:  "Call a service e.g micro call greeter Say.Hello '{\"name\": \"John\"}",
			Action: printer(callService),
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "address",
					Usage:  "Set the address of the service instance to call",
					EnvVar: "MICRO_ADDRESS",
				},
				cli.StringFlag{
					Name:   "output, o",
					Usage:  "Set the output format; json (default), raw",
					EnvVar: "MICRO_OUTPUT",
				},
				cli.StringSliceFlag{
					Name:   "metadata",
					Usage:  "A list of key-value pairs to be forwarded as metadata",
					EnvVar: "MICRO_METADATA",
				},
			},
		},
		{
			Name:   "stream",
			Usage:  "Create a service stream",
			Action: printer(streamService),
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "output, o",
					Usage:  "Set the output format; json (default), raw",
					EnvVar: "MICRO_OUTPUT",
				},
				cli.StringSliceFlag{
					Name:   "metadata",
					Usage:  "A list of key-value pairs to be forwarded as metadata",
					EnvVar: "MICRO_METADATA",
				},
			},
		},
		{
			Name:   "publish",
			Usage:  "Publish a message to a topic",
			Action: printer(publish),
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:   "metadata",
					Usage:  "A list of key-value pairs to be forwarded as metadata",
					EnvVar: "MICRO_METADATA",
				},
			},
		},
	}

	return commands
}
