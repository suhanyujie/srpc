package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/suhanyujie/srpc/pkg/libs/uuid"

	"github.com/suhanyujie/srpc/core"
	"github.com/suhanyujie/srpc/srpc"
)

type Call struct {
	TraceId string
	Method  string
	Args    interface{}
	Reply   interface{}
	Error   error
	Done    chan *Call
}

func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	coder    core.CodeIf
	opt      *srpc.Option
	lock     sync.Mutex // 防止并发造成的字节流混乱
	header   core.Header
	mu       sync.Mutex
	traceId  string
	pending  map[string]*Call
	closing  bool
	shutdown bool
}

var ErrShutdown = errors.New("conn is shutdown")

func (c Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closing {
		return ErrShutdown
	}
	c.closing = true

	return c.coder.Close()
}

func (c *Client) IsAvailable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return !c.shutdown && !c.closing
}

var _ io.Closer = (*Client)(nil)

func (c *Client) registerCall(call *Call) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closing || c.shutdown {
		return "", ErrShutdown
	}
	call.TraceId = c.traceId
	c.pending[call.TraceId] = call
	c.traceId = uuid.NewUuid()

	return call.TraceId, nil
}

func (c *Client) removeCall(traceId string) *Call {
	c.mu.Lock()
	defer c.mu.Unlock()
	call := c.pending[traceId]
	delete(c.pending, traceId)

	return call
}

func (c *Client) terminateCalls(err error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shutdown = true
	for _, call := range c.pending {
		call.Error = err
		call.done()
	}
}

func (c *Client) receive() {
	var err error
	for err == nil {
		var h core.Header
		if err = c.coder.ReadHeader(&h); err != nil {
			break
		}
		call := c.removeCall(h.TraceId)
		switch {
		case call == nil:
			err = c.coder.ReadBody(nil)
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = c.coder.ReadBody(nil)
			call.done()
		default:
			err = c.coder.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	c.terminateCalls(err)
}

func (c *Client) send(call *Call) {
	c.lock.Lock()
	defer c.lock.Unlock()

	newTraceId, err := c.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	c.header.Method = call.Method
	c.header.TraceId = newTraceId
	c.header.Error = ""

	if err := c.coder.Write(&c.header, call.Args); err != nil {
		call := c.removeCall(newTraceId)
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

func (c *Client) Do(method string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panicf("[Do] rpc client: done channel is unbuffered")
	}
	call := &Call{
		Method: method,
		Args:   args,
		Reply:  reply,
		Done:   done,
	}
	c.send(call)

	return call
}

func (c *Client) Call(method string, args, reply interface{}) error {
	call := <-c.Do(method, args, reply, make(chan *Call, 1)).Done

	return call.Error
}

func NewClient(conn net.Conn, opt *srpc.Option) (*Client, error) {
	coderFunc := core.CodeFuncMap[opt.CodeType]
	if coderFunc == nil {
		err := fmt.Errorf("[NewClient] invalid coder type")
		return nil, err
	}
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Printf("[NewClient] rpc client encode options error: %v", err)
		_ = conn.Close()
		return nil, err
	}

	return newClientCoder(coderFunc(conn), opt), nil
}

func newClientCoder(coder core.CodeIf, opt *srpc.Option) *Client {
	client := &Client{
		traceId: uuid.NewUuid(),
		coder:   coder,
		opt:     opt,
		pending: make(map[string]*Call),
	}
	go client.receive()

	return client
}

func Dial(network, address string, opts ...*srpc.Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()

	return NewClient(conn, opt)
}

func parseOptions(opts ...*srpc.Option) (*srpc.Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return srpc.DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("too many options")
	}
	opt := opts[0]
	opt.MagicNumber = srpc.MagicNumber
	if opt.CodeType == "" {
		opt.CodeType = srpc.DefaultOption.CodeType
	}

	return opt, nil
}
