package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/micro/cli"
	"github.com/micro/go-micro/client"
	cbytes "github.com/micro/go-micro/codec/bytes"
	"github.com/micro/go-micro/config/cmd"
	clic "github.com/micro/micro/internal/command/cli"
)

type exec func(*cli.Context, []string) ([]byte, error)

func printer(e exec) func(*cli.Context) {
	return func(c *cli.Context) {
		rsp, err := e(c, c.Args())
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("%s\n", string(rsp))
	}
}

func list(c *cli.Context, args []string) ([]byte, error) {
	// no args

	return nil, errors.New("unknown command")
}

func callService(c *cli.Context, args []string) ([]byte, error) {
	return clic.CallService(c, args)
}

// TODO: stream via HTTP
func streamService(c *cli.Context, args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, errors.New("require service and endpoint")
	}
	service := args[0]
	endpoint := args[1]
	var request map[string]interface{}

	// ignore error
	json.Unmarshal([]byte(strings.Join(args[2:], " ")), &request)

	req := (*cmd.DefaultOptions().Client).NewRequest(service, endpoint, request, client.WithContentType("application/json"))
	stream, err := (*cmd.DefaultOptions().Client).Stream(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("error calling %s.%s: %v", service, endpoint, err)
	}

	if err := stream.Send(request); err != nil {
		return nil, fmt.Errorf("error sending to %s.%s: %v", service, endpoint, err)
	}

	output := c.String("output")

	for {
		if output == "raw" {
			rsp := cbytes.Frame{}
			if err := stream.Recv(&rsp); err != nil {
				return nil, fmt.Errorf("error receiving from %s.%s: %v", service, endpoint, err)
			}
			fmt.Print(string(rsp.Data))
		} else {
			var response map[string]interface{}
			if err := stream.Recv(&response); err != nil {
				return nil, fmt.Errorf("error receiving from %s.%s: %v", service, endpoint, err)
			}
			b, _ := json.MarshalIndent(response, "", "\t")
			fmt.Print(string(b))
		}
	}
}

func publish(c *cli.Context, args []string) ([]byte, error) {
	if err := clic.Publish(c, args); err != nil {
		return nil, err
	}
	return []byte(`ok`), nil
}

func queryHealth(c *cli.Context, args []string) ([]byte, error) {
	return clic.QueryHealth(c, args)
}
