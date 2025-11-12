package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError provides error information returned by the OpenAI API.
// InnerError struct is only valid for Azure OpenAI Service.
type APIError struct {
	Code           any         `json:"code,omitempty"`
	Message        string      `json:"message"`
	Param          *string     `json:"param,omitempty"`
	Type           string      `json:"type"`
	HTTPStatus     string      `json:"-"`
	HTTPStatusCode int         `json:"-"`
	HTTPHeader     http.Header `json:"-"`
	InnerError     *InnerError `json:"innererror,omitempty"`
}

// InnerError Azure Content filtering. Only valid for Azure OpenAI Service.
type InnerError struct {
	Code                 string               `json:"code,omitempty"`
	ContentFilterResults ContentFilterResults `json:"content_filter_result,omitempty"`
}

// RequestError provides information about generic request errors.
type RequestError struct {
	HTTPStatus     string
	HTTPStatusCode int
	Err            error
	Body           []byte
}

type ErrorResponse struct {
	Error *APIError `json:"error,omitempty"`
}

func (e *APIError) Error() string {
	// 创建一个包含所有字段的 map，包括那些 json tag 为 `-` 的字段
	errorMap := map[string]interface{}{
		"http_header":      e.HTTPHeader,
		"code":             e.Code,
		"message":          e.Message,
		"param":            e.Param,
		"type":             e.Type,
		"http_status":      e.HTTPStatus,
		"http_status_code": e.HTTPStatusCode,
		"inner_error":      e.InnerError,
	}

	// 序列化为 JSON 字符串
	jsonBytes, err := json.Marshal(errorMap)
	if err != nil {
		// 如果序列化失败，返回一个简单的错误信息
		return fmt.Sprintf("APIError: status code: %d status: %s, message: %s (failed to marshal: %v)", e.HTTPStatusCode, e.HTTPStatus, e.Message, err)
	}

	return string(jsonBytes)
}

func (e *APIError) UnmarshalJSON(data []byte) (err error) {
	var rawMap map[string]json.RawMessage
	err = json.Unmarshal(data, &rawMap)
	if err != nil {
		return
	}

	err = json.Unmarshal(rawMap["message"], &e.Message)
	if err != nil {
		// If the parameter field of a function call is invalid as a JSON schema
		// refs: https://github.com/sashabaranov/go-openai/issues/381
		var messages []string
		err = json.Unmarshal(rawMap["message"], &messages)
		if err != nil {
			return
		}
		e.Message = strings.Join(messages, ", ")
	}

	// optional fields for azure openai
	// refs: https://github.com/sashabaranov/go-openai/issues/343
	if _, ok := rawMap["type"]; ok {
		err = json.Unmarshal(rawMap["type"], &e.Type)
		if err != nil {
			return
		}
	}

	if _, ok := rawMap["innererror"]; ok {
		err = json.Unmarshal(rawMap["innererror"], &e.InnerError)
		if err != nil {
			return
		}
	}

	// optional fields
	if _, ok := rawMap["param"]; ok {
		err = json.Unmarshal(rawMap["param"], &e.Param)
		if err != nil {
			return
		}
	}

	if _, ok := rawMap["code"]; !ok {
		return nil
	}

	// if the api returned a number, we need to force an integer
	// since the json package defaults to float64
	var intCode int
	err = json.Unmarshal(rawMap["code"], &intCode)
	if err == nil {
		e.Code = intCode
		return nil
	}

	return json.Unmarshal(rawMap["code"], &e.Code)
}

func (e *RequestError) Error() string {
	return fmt.Sprintf(
		"error, status code: %d, status: %s, message: %s, body: %s",
		e.HTTPStatusCode, e.HTTPStatus, e.Err, e.Body,
	)
}

func (e *RequestError) Unwrap() error {
	return e.Err
}
