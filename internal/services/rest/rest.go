package rest

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultTimeout = 30 * time.Second

var ErrUnexpectedStatusCode = errors.New("unexpected status code")

type Request struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    string
	Timeout time.Duration
}

type Response struct {
	Status  string      `expr:"status"`
	Code    int         `expr:"code"`
	Body    string      `expr:"body"`
	Headers http.Header `expr:"headers"`
}

func SendRequest(reqSpec *Request, validStatusCodes []int) (*Response, error) {
	setRequestDefaults(reqSpec)
	client := http.Client{Timeout: reqSpec.Timeout}
	reqURL, err := url.Parse(reqSpec.URL)
	if err != nil {
		return nil, err
	}

	headers := make(http.Header)
	for k, v := range reqSpec.Headers {
		headers.Add(k, v)
	}
	req := http.Request{
		Method: strings.ToUpper(reqSpec.Method),
		URL:    reqURL,
		Header: headers,
	}
	if reqSpec.Body != "" {
		req.Body = io.NopCloser(strings.NewReader(reqSpec.Body))
	}

	httpResp, err := client.Do(&req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if !isStatusCodeAccepted(httpResp.StatusCode, validStatusCodes) {
		return nil, ErrUnexpectedStatusCode
	}

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	resp := &Response{
		Status:  httpResp.Status,
		Code:    httpResp.StatusCode,
		Body:    string(respBody),
		Headers: httpResp.Header,
	}
	return resp, nil
}

func setRequestDefaults(req *Request) {
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.Timeout == 0 {
		req.Timeout = DefaultTimeout
	}
}

func isStatusCodeAccepted(statusCode int, acceptedStatusCodes []int) bool {
	if len(acceptedStatusCodes) == 0 {
		return statusCode >= 200 && statusCode < 300
	}
	for _, acceptedStatusCode := range acceptedStatusCodes {
		if statusCode == acceptedStatusCode {
			return true
		}
	}
	return false
}
