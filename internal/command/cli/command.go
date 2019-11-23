package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/micro/cli"
	"github.com/micro/go-micro/client"
	cbytes "github.com/micro/go-micro/codec/bytes"
	"github.com/micro/go-micro/config/cmd"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/registry"

	proto "github.com/micro/go-micro/debug/proto"

	"github.com/serenize/snaker"
)

func formatEndpoint(v *registry.Value, r int) string {
	// default format is tabbed plus the value plus new line
	fparts := []string{"", "%s %s", "\n"}
	for i := 0; i < r+1; i++ {
		fparts[0] += "\t"
	}
	// its just a primitive of sorts so return
	if len(v.Values) == 0 {
		return fmt.Sprintf(strings.Join(fparts, ""), snaker.CamelToSnake(v.Name), v.Type)
	}

	// this thing has more things, it's complex
	fparts[1] += " {"

	vals := []interface{}{snaker.CamelToSnake(v.Name), v.Type}

	for _, val := range v.Values {
		fparts = append(fparts, "%s")
		vals = append(vals, formatEndpoint(val, r+1))
	}

	// at the end
	l := len(fparts) - 1
	for i := 0; i < r+1; i++ {
		fparts[l] += "\t"
	}
	fparts = append(fparts, "}\n")

	return fmt.Sprintf(strings.Join(fparts, ""), vals...)
}

func del(url string, b []byte, v interface{}) error {
	if !strings.HasPrefix(url, "http") && !strings.HasPrefix(url, "https") {
		url = "http://" + url
	}

	buf := bytes.NewBuffer(b)
	defer buf.Reset()

	req, err := http.NewRequest("DELETE", url, buf)
	if err != nil {
		return err
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if v == nil {
		return nil
	}

	d := json.NewDecoder(rsp.Body)
	d.UseNumber()
	return d.Decode(v)
}

func get(url string, v interface{}) error {
	if !strings.HasPrefix(url, "http") && !strings.HasPrefix(url, "https") {
		url = "http://" + url
	}

	rsp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	d := json.NewDecoder(rsp.Body)
	d.UseNumber()
	return d.Decode(v)
}

func post(url string, b []byte, v interface{}) error {
	if !strings.HasPrefix(url, "http") && !strings.HasPrefix(url, "https") {
		url = "http://" + url
	}

	buf := bytes.NewBuffer(b)
	defer buf.Reset()

	rsp, err := http.Post(url, "application/json", buf)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if v == nil {
		return nil
	}

	d := json.NewDecoder(rsp.Body)
	d.UseNumber()
	return d.Decode(v)
}

func getPeers(v map[string]interface{}) map[string]string {
	if v == nil {
		return nil
	}

	peers := make(map[string]string)
	node := v["node"].(map[string]interface{})
	peers[node["id"].(string)] = node["address"].(string)

	// return peers if nil
	if v["peers"] == nil {
		return peers
	}

	nodes := v["peers"].([]interface{})

	for _, peer := range nodes {
		p := getPeers(peer.(map[string]interface{}))
		for id, address := range p {
			peers[id] = address
		}
	}

	return peers
}

func callContext(c *cli.Context) context.Context {
	callMD := make(map[string]string)

	for _, md := range c.StringSlice("metadata") {
		parts := strings.Split(md, "=")
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		val := strings.Join(parts[1:], "=")

		// set the key/val
		callMD[key] = val
	}

	return metadata.NewContext(context.Background(), callMD)
}

func Publish(c *cli.Context, args []string) error {
	if len(args) < 2 {
		return errors.New("require topic and message e.g micro publish event '{\"hello\": \"world\"}'")
	}
	defer func() {
		time.Sleep(time.Millisecond * 100)
	}()
	topic := args[0]
	message := args[1]

	cl := *cmd.DefaultOptions().Client
	ct := func(o *client.MessageOptions) {
		o.ContentType = "application/json"
	}

	d := json.NewDecoder(strings.NewReader(message))
	d.UseNumber()

	var msg map[string]interface{}
	if err := d.Decode(&msg); err != nil {
		return err
	}

	ctx := callContext(c)
	m := cl.NewMessage(topic, msg, ct)
	return cl.Publish(ctx, m)
}

func CallService(c *cli.Context, args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, errors.New(`require service and endpoint e.g micro call greeeter Say.Hello '{"name": "john"}'`)
	}

	var req, service, endpoint string
	service = args[0]
	endpoint = args[1]

	if len(args) > 2 {
		req = strings.Join(args[2:], " ")
	}

	// empty request
	if len(req) == 0 {
		req = `{}`
	}

	var request map[string]interface{}
	var response []byte

	d := json.NewDecoder(strings.NewReader(req))
	d.UseNumber()

	if err := d.Decode(&request); err != nil {
		return nil, err
	}

	ctx := callContext(c)
	creq := (*cmd.DefaultOptions().Client).NewRequest(service, endpoint, request, client.WithContentType("application/json"))

	var opts []client.CallOption

	if addr := c.String("address"); len(addr) > 0 {
		opts = append(opts, client.WithAddress(addr))
	}

	var err error
	if output := c.String("output"); output == "raw" {
		rsp := cbytes.Frame{}
		err = (*cmd.DefaultOptions().Client).Call(ctx, creq, &rsp, opts...)
		// set the raw output
		response = rsp.Data
	} else {
		var rsp json.RawMessage
		err = (*cmd.DefaultOptions().Client).Call(ctx, creq, &rsp, opts...)
		// set the response
		if err == nil {
			var out bytes.Buffer
			defer out.Reset()
			if err := json.Indent(&out, rsp, "", "\t"); err != nil {
				return nil, err
			}
			response = out.Bytes()
		}
	}

	if err != nil {
		return nil, fmt.Errorf("error calling %s.%s: %v", service, endpoint, err)
	}

	return response, nil
}

func QueryHealth(c *cli.Context, args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, errors.New("require service name")
	}

	req := (*cmd.DefaultOptions().Client).NewRequest(args[0], "Debug.Health", &proto.HealthRequest{})

	// if the address is specified then we just call it
	if addr := c.String("address"); len(addr) > 0 {
		rsp := &proto.HealthResponse{}
		err := (*cmd.DefaultOptions().Client).Call(
			context.Background(),
			req,
			rsp,
			client.WithAddress(addr),
		)
		if err != nil {
			return nil, err
		}
		return []byte(rsp.Status), nil
	}

	// otherwise get the service and call each instance individually
	service, err := (*cmd.DefaultOptions().Registry).GetService(args[0])
	if err != nil {
		return nil, err
	}

	if len(service) == 0 {
		return nil, errors.New("Service not found")
	}

	var output []string

	// print things
	output = append(output, "service  "+service[0].Name)

	for _, serv := range service {
		// print things
		output = append(output, "\nversion "+serv.Version)
		output = append(output, "\nnode\t\taddress:port\t\tstatus")

		// query health for every node
		for _, node := range serv.Nodes {
			address := node.Address
			rsp := &proto.HealthResponse{}

			var err error

			// call using client
			err = (*cmd.DefaultOptions().Client).Call(
				context.Background(),
				req,
				rsp,
				client.WithAddress(address),
			)

			var status string
			if err != nil {
				status = err.Error()
			} else {
				status = rsp.Status
			}
			output = append(output, fmt.Sprintf("%s\t\t%s\t\t%s", node.Id, node.Address, status))
		}
	}

	return []byte(strings.Join(output, "\n")), nil
}
