package t38c

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// Client allows you to interact with the Tile38 server.
type Client struct {
	debug    bool
	password *string
	executor Executor
}

// ClientOption ...
type ClientOption func(*Client)

// Debug option.
func Debug() ClientOption {
	return func(c *Client) {
		c.debug = true
	}
}

// WithPassword option.
func WithPassword(password string) ClientOption {
	return func(c *Client) {
		c.password = &password
	}
}

// New creates a new Tile38 client.
// By default uses redis pool with 5 connections.
// In debug mode will also print commands which will be sent to the server.
func New(addr string, opts ...ClientOption) (*Client, error) {
	dialer := NewRadixPool(addr, 5)
	return NewWithDialer(dialer, opts...)
}

// NewWithDialer creates a new Tile38 client with provided dialer.
// See Executor interface for more information.
func NewWithDialer(dialer ExecutorDialer, opts ...ClientOption) (*Client, error) {
	client := &Client{}
	for _, opt := range opts {
		opt(client)
	}

	executor, err := dialer(client.password)
	if err != nil {
		return nil, err
	}

	client.executor = executor
	if err := client.Ping(); err != nil {
		return nil, err
	}

	return client, nil
}

func (client *Client) executeCmd(cmd Command) ([]byte, error) {
	resp, err := client.executor.Execute(cmd.Name, cmd.Args...)
	if client.debug {
		log.Printf("[%s]: %s", cmd, resp)
	}

	if err != nil {
		return nil, err
	}

	if err := checkResponseErr(resp); err != nil {
		return nil, fmt.Errorf("command: %s: %v", cmd, err)
	}

	return resp, nil
}

func (client *Client) jExecute(resp interface{}, command string, args ...string) error {
	b, err := client.Execute(command, args...)
	if err != nil {
		return err
	}

	if resp != nil {
		return json.Unmarshal(b, &resp)
	}

	return nil
}

// Execute Tile38 command.
func (client *Client) Execute(command string, args ...string) ([]byte, error) {
	return client.executeCmd(NewCommand(command, args...))
}

// ExecuteStream used for Tile38 commands with streaming response.
func (client *Client) ExecuteStream(ctx context.Context, command string, args ...string) (chan []byte, error) {
	if client.debug {
		status := "ok"
		ch, err := client.executor.ExecuteStream(ctx, command, args...)
		if err != nil {
			status = err.Error()
		}

		log.Printf("[%s]: %s", NewCommand(command, args...), status)
		return ch, err
	}

	return client.executor.ExecuteStream(ctx, command, args...)
}
