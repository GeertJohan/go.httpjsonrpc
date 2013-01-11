package httpjsonrpc

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
)

type Client struct {
	// remote host url
	url string

	// http.Client
	httpClient *http.Client

	// basic auth
	username string
	password string

	// sequence id
	id     uint64
	idLock sync.Mutex
}

// Create new Client instance
func NewClient(url string) *Client {
	return &Client{
		httpClient: &http.Client{},
		url:        url,
	}
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

func (c *Client) Call(method string, params interface{}, reply interface{}) error {
	// create request object
	reqObj := new(Request)
	reqObj.Method = method
	reqObj.Params = params
	reqObj.Id = c.newId()

	// encode request to buffer
	bufSend := &bytes.Buffer{}
	enc := json.NewEncoder(bufSend)
	err := enc.Encode(reqObj)
	if err != nil {
		return err
	}

	// create http request
	req, err := http.NewRequest("POST", c.url, bufSend)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if len(c.username) > 0 {

	}

	// do request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// unmarshall response
	bufRecv, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bufRecv, reply)
	if err != nil {
		return err
	}

	// done
	return nil
}
