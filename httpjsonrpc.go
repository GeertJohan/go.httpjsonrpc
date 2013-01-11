package httpjsonrpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
)

type Client struct {
	// remote host url
	url string

	// http.Client
	httpClient *http.Client

	// basic auth
	basicAuthUsername string
	basicAuthPassword string

	// sequence id
	id     uint64
	idLock sync.Mutex
}

// Create new Client instance
func NewClient(url string, httpClient *http.Client) *Client {
	return &Client{
		httpClient: httpClient,
		url:        url,
	}
}

func (c *Client) SetBasicAuth(username, password string) {
	c.basicAuthPassword = username
	c.basicAuthPassword = password
}

func (c *Client) newId() (id uint64) {
	c.idLock.Lock()
	id = c.id
	c.id++
	c.idLock.Unlock()
	return id
}

type Request struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
	Id     uint64      `json:"id"`
}

type Response struct {
	Id        uint64           `json:"id"`
	RawResult *json.RawMessage `json:"result"`
	Error     interface{}      `json:"error"`
}

func (c *Client) Call(method string, params interface{}, result interface{}) (response *Response, err error) {
	// create request object
	reqObj := new(Request)
	reqObj.Method = method
	reqObj.Params = params
	reqObj.Id = c.newId()

	// encode request to buffer
	bufSend := &bytes.Buffer{}
	enc := json.NewEncoder(bufSend)
	err = enc.Encode(reqObj)
	if err != nil {
		return nil, err
	}

	// create http request
	req, err := http.NewRequest("POST", c.url, bufSend)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if len(c.basicAuthUsername) > 0 {
		req.SetBasicAuth(c.basicAuthUsername, c.basicAuthPassword)
	}

	// do request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// unmarshall response
	respObj := new(Response)
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(respObj)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(*respObj.RawResult), result)
	if err != nil {
		return nil, err
	}

	// done
	return respObj, nil
}
