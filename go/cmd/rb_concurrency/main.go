package main

import (
	"C"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

//export Proc
func Proc(encoded_json *C.char) *C.char {
	message := C.GoString(encoded_json)

	result, err := process(message)
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "Error: %v"}`, err))
	}

	return C.CString(string(result))
}

func process(message string) ([]byte, error) {
	var data []json.RawMessage
	if err := json.Unmarshal([]byte(message), &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	wg := sync.WaitGroup{}
	mux := sync.Mutex{}

	client := &http.Client{}
	ctx := context.Background()
	wrapRubyRequest := make([]*WrapRubyRequest, len(data))
	for i, item := range data {
		var req RubyRequest
		if err := json.Unmarshal(item, &req); err != nil {
			return nil, fmt.Errorf("error unmarshalling request: %v", err)
		}
		wrapRubyRequest[i] = &WrapRubyRequest{
			RubyRequest: &req,
			index:       i,
		}
	}

	results := make([]*RubyResponse, len(data))
	for _, item := range wrapRubyRequest {
		wg.Add(1)
		func(ctx context.Context, client *http.Client, item *WrapRubyRequest) {
			defer wg.Done()
			resp := request(ctx, client, item.RubyRequest)
			mux.Lock()
			results[item.index] = resp
			mux.Unlock()
		}(ctx, client, item)
	}
	wg.Wait()

	result, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("error marshalling results: %v", err)
	}

	return result, nil
}

type RubyRequest struct {
	Method  string              `json:"method"`
	URI     string              `json:"uri"`
	Headers map[string][]string `json:"headers"`
	Body    *NullableRawMessage `json:"body,omitempty"`
}

type WrapRubyRequest struct {
	RubyRequest *RubyRequest
	index       int
}

type NullableRawMessage json.RawMessage

func (n *NullableRawMessage) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*n = nil
		return nil
	}
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("error unmarshalling NullableRawMessage: %v", err)
	}
	*n = NullableRawMessage(raw)
	return nil
}

type RubyResponse struct {
	Status      int                 `json:"status"`
	Headers     map[string][]string `json:"headers"`
	Body        string              `json:"body,omitempty"`
	RubyRequest *RubyRequest        `json:"request,omitempty"`
}

func request(ctx context.Context, client *http.Client, req *RubyRequest) *RubyResponse {
	var body io.Reader
	if req.Body != nil {
		body = bytes.NewBuffer(*req.Body)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, req.Method, req.URI, body)
	if err != nil {
		return &RubyResponse{Status: http.StatusInternalServerError, Body: fmt.Sprintf(`{"error": "Error creating request: %v"}`, err)}
	}

	for k, headers := range req.Headers {
		for _, v := range headers {
			httpRequest.Header.Set(k, v)
		}
	}

	resp, err := client.Do(httpRequest.WithContext(ctx))
	if err != nil {
		return &RubyResponse{Status: http.StatusInternalServerError, Body: fmt.Sprintf(`{"error": "Error creating request: %v"}`, err)}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &RubyResponse{Status: http.StatusInternalServerError, Body: fmt.Sprintf(`{"error": "Error creating request: %v"}`, err)}
	}

	b := json.RawMessage(bodyBytes)
	return &RubyResponse{
		Status:      resp.StatusCode,
		Headers:     resp.Header,
		Body:        string(b),
		RubyRequest: req,
	}
}

func main() {}

const exampleJSON = ""
