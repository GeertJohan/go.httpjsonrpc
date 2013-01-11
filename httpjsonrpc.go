package httpjsonrpc

// This package is modified from the the jsonrpc package (golang.org/pkg/net/rpc/jsonrpc/) to perform http json-rpc.

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"sync"
)

type clientCodec struct {
	dec *json.Decoder
	enc *json.Encoder

	// http.Request and http.Client to use for communication
	// TODO: Maybe make an empty post request once in a while to check for notifications?
	httpReq    *http.Request
	httpClient *http.Client

	// temporary work space
	req  clientRequest
	resp clientResponse

	// JSON-RPC responses include the request id but not the request method.
	// Package rpc expects both.
	// We save the request method in pending when sending a request
	// and then look it up by request ID when filling out the rpc Response.
	mutex   sync.Mutex           // protects pending
	pending map[uint64]string    // map request id to method name
	results map[uint64]io.Reader // map request id to result buffer
}

// NewClientCodec returns a new rpc.ClientCodec using JSON-RPC on http.Client.
func NewClientCodec(httpClient *http.Client, url string) rpc.ClientCodec {
	bufIn := bytes.Buffer{}
	bufOut := bytes.Buffer{}

	return &clientCodec{
		dec:        json.NewDecoder(bufIn),
		enc:        json.NewEncoder(bufOut),
		httpReq:    http.NewRequest("post", url, bufOut), // TODO: Is "application/json" required?
		httpClient: httpClient,
		pending:    make(map[uint64]string),
	}
}

type clientRequest struct {
	Method string         `json:"method"`
	Params [1]interface{} `json:"params"`
	Id     uint64         `json:"id"`
}

func (c *clientCodec) WriteRequest(r *rpc.Request, param interface{}) error {
	c.mutex.Lock()
	c.pending[r.Seq] = r.ServiceMethod
	c.mutex.Unlock()
	c.req.Method = r.ServiceMethod
	c.req.Params[0] = param
	c.req.Id = r.Seq
	err := c.enc.Encode(&c.req)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(c.httpReq)
}

type clientResponse struct {
	Id     uint64           `json:"id"`
	Result *json.RawMessage `json:"result"`
	Error  interface{}      `json:"error"`
}

func (r *clientResponse) reset() {
	r.Id = 0
	r.Result = nil
	r.Error = nil
}

func (c *clientCodec) ReadResponseHeader(r *rpc.Response) error {
	c.resp.reset()
	if err := c.dec.Decode(&c.resp); err != nil {
		return err
	}

	c.mutex.Lock()
	r.ServiceMethod = c.pending[c.resp.Id]
	delete(c.pending, c.resp.Id)
	c.mutex.Unlock()

	r.Error = ""
	r.Seq = c.resp.Id
	if c.resp.Error != nil {
		x, ok := c.resp.Error.(string)
		if !ok {
			return fmt.Errorf("invalid error %v", c.resp.Error)
		}
		if x == "" {
			x = "unspecified error"
		}
		r.Error = x
	}
	return nil
}

func (c *clientCodec) ReadResponseBody(x interface{}) error {
	if x == nil {
		return nil
	}
	return json.Unmarshal(*c.resp.Result, x)
}

func (c *clientCodec) Close() error {
	return c.c.Close()
}

// NewClient returns a new rpc.Client to handle requests to the
// set of services at the other end of the connection.
func NewClient(url string) *rpc.Client {
	cli := &http.Client{}
	return rpc.NewClientWithCodec(NewClientCodec(cli, url))
}
