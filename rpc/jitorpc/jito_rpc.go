package jitorpc
import(
  "context"
  "net/http" 
  "encoding/json"
  "fmt"
  "bytes"
  "sync/atomic"
  "errors"
  "reflect"
  
  "github.com/davecgh/go-spew/spew"
)

type RPCClient interface{
  MakeCall(ctx context.Context, path string, RPCPayload *RPCPayload) (*RPCResponse, error)
  MakeCallWithHeader(ctx context.Context, path string, RPCPayload *RPCPayload) (*RPCResponseWithHeader, error)
} 

type HTTPClient interface{
  Do(*http.Request) (*http.Response, error)
}

type rpcClient struct {
	endpoint      string
	httpClient    HTTPClient
}

type RPCPayload struct {
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      any         `json:"id"`
	JSONRPC string      `json:"jsonrpc"`
}

type RPCResponse struct {
	JSONRPC string             `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError          `json:"error,omitempty"`
	ID      any                `json:"id"`
}

type RPCResponseWithHeader struct {
	JSONRPC string             `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError          `json:"error,omitempty"`
	ID      any                `json:"id"`
  BundleID string
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var spewConf = spew.ConfigState{
	Indent:                " ",
	DisableMethods:        true,
	DisablePointerMethods: true,
	SortKeys:              true,
}

type HTTPError struct {
	Code int
	err  error
}

func (e *HTTPError) Error() string {
	return e.err.Error()
}

// HTTPClient is an abstraction for a HTTP client

// Error function is provided to be used as error object.
func (e *RPCError) Error() string {
	return spewConf.Sdump(e)
}

func NewClient(endpoint string) RPCClient {
	return &rpcClient{
		endpoint:    endpoint,
		httpClient:   &http.Client{},
	}
}

func (client *rpcClient) MakeCall(ctx context.Context, path string, RPCPayload *RPCPayload,
) (*RPCResponse, error){
	var rpcResponse *RPCResponse
  // could get http.Response and err
	_, err := client.doCallWithCallbackOnHTTPResponse(
		ctx,
		RPCPayload,
		func(httpRequest *http.Request, httpResponse *http.Response) error {
			decoder := json.NewDecoder(httpResponse.Body)
			decoder.DisallowUnknownFields()
			decoder.UseNumber()
			err := decoder.Decode(&rpcResponse)
			// parsing error
			if err != nil {
				// if we have some http error, return it
				if httpResponse.StatusCode >= 400 {
					return &HTTPError{
						Code: httpResponse.StatusCode,
						err:  fmt.Errorf("rpc call %v() on %v status code: %v. could not decode body to rpc response: %w", RPCPayload.Method, httpRequest.URL.String(), httpResponse.StatusCode, err),
					}
				}
				return fmt.Errorf("rpc call %v() on %v status code: %v. could not decode body to rpc response: %w", RPCPayload.Method, httpRequest.URL.String(), httpResponse.StatusCode, err)
			}

			// response body empty
			if rpcResponse == nil {
				// if we have some http error, return it
				if httpResponse.StatusCode >= 400 {
					return &HTTPError{
						Code: httpResponse.StatusCode,
						err:  fmt.Errorf("rpc call %v() on %v status code: %v. rpc response missing", RPCPayload.Method, httpRequest.URL.String(), httpResponse.StatusCode),
					}
				}
				return fmt.Errorf("rpc call %v() on %v status code: %v. rpc response missing", RPCPayload.Method, httpRequest.URL.String(), httpResponse.StatusCode)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return rpcResponse, nil
}


func (client *rpcClient) MakeCallWithHeader(ctx context.Context, path string, RPCPayload *RPCPayload,
) (*RPCResponseWithHeader, error) {
	var rpcResponse *RPCResponseWithHeader
  // could get http.Response and err
	httpResp, err := client.doCallWithCallbackOnHTTPResponse(
		ctx,
		RPCPayload,
		func(httpRequest *http.Request, httpResponse *http.Response) error {
			decoder := json.NewDecoder(httpResponse.Body)
			decoder.DisallowUnknownFields()
			decoder.UseNumber()
			err := decoder.Decode(&rpcResponse)
			// parsing error
			if err != nil {
				// if we have some http error, return it
				if httpResponse.StatusCode >= 400 {
					return &HTTPError{
						Code: httpResponse.StatusCode,
						err:  fmt.Errorf("rpc call %v() on %v status code: %v. could not decode body to rpc response: %w", RPCPayload.Method, httpRequest.URL.String(), httpResponse.StatusCode, err),
					}
				}
				return fmt.Errorf("rpc call %v() on %v status code: %v. could not decode body to rpc response: %w", RPCPayload.Method, httpRequest.URL.String(), httpResponse.StatusCode, err)
			}

			// response body empty
			if rpcResponse == nil {
				// if we have some http error, return it
				if httpResponse.StatusCode >= 400 {
					return &HTTPError{
						Code: httpResponse.StatusCode,
						err:  fmt.Errorf("rpc call %v() on %v status code: %v. rpc response missing", RPCPayload.Method, httpRequest.URL.String(), httpResponse.StatusCode),
					}
				}
				return fmt.Errorf("rpc call %v() on %v status code: %v. rpc response missing", RPCPayload.Method, httpRequest.URL.String(), httpResponse.StatusCode)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
  
  rpcResponse.BundleID = httpResp.Header.Get("x_bundle_id")

	return rpcResponse, nil
}

func (client *rpcClient) doCallWithCallbackOnHTTPResponse(
	ctx context.Context,
	RPCPayload *RPCPayload,
	callback func(*http.Request, *http.Response) error,
) (*http.Response, error) {
	if RPCPayload != nil && RPCPayload.ID == nil {
		RPCPayload.ID = newID()
	}
	httpRequest, err := client.newRequest(ctx, RPCPayload)
	if err != nil {
		if httpRequest != nil {
			return nil, fmt.Errorf("rpc call %v() on %v: %w", RPCPayload.Method, httpRequest.URL.String(), err)
		}
		return nil, fmt.Errorf("rpc call %v(): %w", RPCPayload.Method, err)
	}
	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("rpc call %v() on %v: %w", RPCPayload.Method, httpRequest.URL.String(), err)
	}
	defer httpResponse.Body.Close()

	return httpResponse, callback(httpRequest, httpResponse)
}

func (client *rpcClient) newRequest(ctx context.Context, req interface{}) (*http.Request, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", client.endpoint, bytes.NewReader(body))
	if err != nil {
		return request, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")


	return request, nil
}

var useIntegerID = false
var integerID = new(atomic.Uint64)

func newID() any {
	if useIntegerID {
		return integerID.Add(1) // gives a unique id
	} else {
		return 1 // fixed id
	}
}

func (RPCResponse *RPCResponse) GetObject(toType interface{}) error {
	if RPCResponse == nil {
		return errors.New("rpc response is nil")
	}
	rv := reflect.ValueOf(toType)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("expected a pointer got a value instead: %v", reflect.TypeOf(toType))
	}
	if RPCResponse.Result == nil {
		RPCResponse.Result = []byte(`null`)
	}

	return json.Unmarshal(RPCResponse.Result, toType)
}
