package httpjsonrpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	basicAuthUsername string
	basicAuthPassword string

	// sequence id
	id     uint64
	idLock sync.Mutex
}

// Create new Client instance
func NewClient(url string, customHttpClient *http.Client) *Client {
	if customHttpClient == nil {
		customHttpClient = &http.Client{}
	}

	return &Client{
		httpClient: customHttpClient,
		url:        url,
	}
}

func (c *Client) SetBasicAuth(username, password string) {
	c.basicAuthUsername = username
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
		return nil, fmt.Errorf("Error when encoding json: %s", err)
	}

	// create http request
	req, err := http.NewRequest("POST", c.url, bufSend)
	if err != nil {
		return nil, fmt.Errorf("Error when creating new http request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if len(c.basicAuthUsername) > 0 {
		req.SetBasicAuth(c.basicAuthUsername, c.basicAuthPassword)
	}

	// do request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error when performing http request: %s", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		break
	case http.StatusUnauthorized:
		return nil, errors.New("HTTP status unauthorized. Probably need valid basicAuth.")
	default:
		return nil, fmt.Errorf("unexpected http status code %d", resp.StatusCode)
	}

	// unmarshall response
	respObj := new(Response)
	buff, err := ioutil.ReadAll(resp.Body)
	fmt.Printf("%s", string(buff))
	return
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(respObj)
	if err != nil {
		return nil, fmt.Errorf("Error when decoding json: %s", err)
	}

	err = json.Unmarshal([]byte(*respObj.RawResult), result)
	if err != nil {
		return nil, fmt.Errorf("Error when unmarshalling result from response: %s", err)
	}

	// done
	return respObj, nil
}
