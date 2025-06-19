package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type FetchRequest struct {
	Method  string
	Url     string
	Headers map[string]string
	Body    io.Reader
	Client  *http.Client
	Ctx     *context.Context
	Async   bool
}

type FetchResponse struct {
	response *http.Response
	Promise  *Promise[*FetchResponse]
	Body     *Thunk[Option[[]byte]]
	Error    error
	Async    bool
}

func ptr[T any](p T) *T {
	return &p
}

func (f *FetchResponse) Response() (*http.Response, error) {
  if f.Error != nil {
    return nil, f.Error
  }
  r := f.Promise.Await().Result.response
  if f.Error != nil {
    return r, f.Error
  }
  if r == nil{
    return nil, fmt.Errorf("nil response")
  }
  return r, nil
}

func (f *FetchResponse) Bytes() ([]byte, error) {
  if f.Error != nil {
    return nil, f.Error
  }
  p := f.Promise.Await()
  if p.Error != nil {
    f.Error = p.Error
    return nil, f.Error
  }
  b := p.Result.Body.Await().Result
  if b.Error != nil {
    f.Error = b.Error
  }
  return b.Result, f.Error
}

func (f *FetchResponse) Text() (string, error) {
	b, e := f.Bytes()
	return string(b), e
}

func (f *FetchResponse) JSON() (interface{}, error) {
  b, e := f.Bytes()
  if e != nil {
    return nil, e
  }
  var v interface{}
  e = json.Unmarshal(b, &v)
  return v, e
}

func main() {
	req := FetchRequest{Url: "https://www.google.com", Async: true}
	res := FetchSuccess(req)
	fmt.Println(res)
	fmt.Println(res.Text())
}

func fetch(req FetchRequest) FetchResponse {
	client := http.DefaultClient
	if req.Client != nil {
		client = req.Client
	}
	if len(req.Method) == 0 {
		req.Method = "GET"
	}
	var request *http.Request
	var err error
	if req.Ctx != nil {
		request, err = http.NewRequestWithContext(*req.Ctx, req.Method, req.Url, req.Body)
	} else {
		request, err = http.NewRequest(req.Method, req.Url, req.Body)
	}
	if err != nil {
		return FetchResponse{Error: err}
	}

	if req.Headers != nil {
		for key, value := range req.Headers {
			request.Header.Add(key, value)
		}
	}

	response, err := client.Do(request)
	if err != nil {
		return FetchResponse{Error: err}
	}
	if response == nil {
		return FetchResponse{Error: fmt.Errorf("nil response")}
	}
	fr := FetchResponse{
		response: response,
		Body: NewThunk(func() Option[[]byte] {
			return NewOption(func() ([]byte, error) {
				return ReadResponseBody(response)
			})()
		}),
	}
	fr.Promise = &Promise[*FetchResponse]{
		Result: &fr,
		Done:   true,
	}
	return fr
}

func fetchAsync(req FetchRequest, fetchSync func(FetchRequest) FetchResponse) FetchResponse {
  fr := FetchResponse{
    Async:true,
  }
	promise := NewPromise(func() *FetchResponse {
		fetchRes := fetchSync(req)
		fetchRes.Async = true
		return &fetchRes
	})
	thunk := NewThunk(func() Option[[]byte] {
		return NewOption(func() ([]byte, error) {
			p := promise.Await()
			if p.Error != nil {
        fr.Error = p.Error
				return nil, p.Error
			}
			r := p.Result
			if r.Error != nil {
        p.Error = r.Error
        fr.Error = r.Error
				return nil, r.Error
			}
			body,err:= ReadResponseBody(r.response)
      if err != nil {
        r.Error = err
        p.Error = err
        fr.Error = err
        return nil,err
      }
      return body,nil
		})()
	})
	fr.Promise = &promise
	fr.Body =    thunk
	fr.Error =   promise.Error
  return fr
}

func Fetch(req FetchRequest) FetchResponse {
	var res FetchResponse
	(func() {
		defer func() {
			if r := recover(); r != nil {
				res.Error = fmt.Errorf("panic in http fetch: %+v", r)
			}
		}()
		if req.Async {
			res = fetchAsync(req, fetch)
		} else {
			res = fetch(req)
		}
	})()
	return res
}

func fetchOK(req FetchRequest) FetchResponse {
	res := fetch(req)
	if res.Error != nil {
		return res
	}
	if res.response == nil {
		res.Error = fmt.Errorf("nil response")
		return res
	}
	if res.response.StatusCode != http.StatusOK {
		defer CloseResponseBody(res.response)
		res.Error = fmt.Errorf("http fetch not OK failed with status %s", res.response.Status)
	}
	return res
}

func FetchOK(req FetchRequest) FetchResponse {
	var res FetchResponse
	(func() {
		defer func() {
			if r := recover(); r != nil {
				res.Error = fmt.Errorf("panic in http fetch: %+v", r)
			}
		}()
		if req.Async {
			res = fetchAsync(req, fetchOK)
		} else {
			res = fetchOK(req)
		}
	})()
	return res
}

func fetchSuccess(req FetchRequest) FetchResponse {
	res := fetch(req)
	if res.Error != nil {
		return res
	}
	if res.response == nil {
		res.Error = fmt.Errorf("nil response")
		return res
	}
	if res.response.StatusCode == 0 || res.response.StatusCode >= 300 {
		defer CloseResponseBody(res.response)
		res.Error = fmt.Errorf("http fetch failed with status %s", res.response.Status)
	}
	return res
}

func FetchSuccess(req FetchRequest) FetchResponse {
	var res FetchResponse
	(func() {
		defer func() {
			if r := recover(); r != nil {
				res.Error = fmt.Errorf("panic in http fetch: %+v", r)
			}
		}()
		if req.Async {
			res = fetchAsync(req, fetchSuccess)
		} else {
			res = fetchSuccess(req)
		}
	})()
	return res
}

func fetchStatus(req FetchRequest, statusCode int) FetchResponse {
	res := fetch(req)
	if res.Error != nil {
		return res
	}
	if res.response == nil {
		res.Error = fmt.Errorf("nil response")
		return res
	}
	if res.response.StatusCode != statusCode {
		defer CloseResponseBody(res.response)
		res.Error = fmt.Errorf("http fetch failed with status %s", res.response.Status)
	}
	return res
}

func FetchStatus(req FetchRequest, statusCode int) FetchResponse {
	var res FetchResponse
	(func() {
		defer func() {
			if r := recover(); r != nil {
				res.Error = fmt.Errorf("panic in http fetch: %+v", r)
			}
		}()
		if req.Async {
			res = fetchAsync(req, func(r FetchRequest) FetchResponse {
				return fetchStatus(r, statusCode)
			})
		} else {
			res = fetchStatus(req, statusCode)
		}
	})()
	return res
}

func CloseBody(body io.ReadCloser) error {
	if body == nil {
		return fmt.Errorf("tried to close a nil body")
	}
	var err error
	(func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in closing body: %+v", r)
			}
		}()
		err = body.Close()
	})()
	return err
}

func CloseResponseBody(res *http.Response) error {
	if res == nil {
		return fmt.Errorf("tried to close a nil response")
	}
	return CloseBody(res.Body)
}

func readBody(body io.ReadCloser) ([]byte, error) {
	if body == nil {
		return nil, fmt.Errorf("tried to read a nil body")
	}
	defer CloseBody(body)
	bits, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return bits, nil
}

func ReadBody(body io.ReadCloser) ([]byte, error) {
	var bits []byte
	var err error
	(func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in reading body: %+v", r)
			}
		}()
		bits, err = readBody(body)
	})()
	return bits, err
}

func ReadResponseBody(res *http.Response) ([]byte, error) {
	if res == nil {
		return nil, fmt.Errorf("tried to read a nil response")
	}
	return ReadBody(res.Body)
}

func FetchBody(req FetchRequest) ([]byte, error) {
	if req.Async {
		return nil, fmt.Errorf("Called synchronous FetchBody with async request")
	}
	res := FetchSuccess(req)
	if res.Error != nil {
		return nil, res.Error
	}
	return ReadResponseBody(res.response)
}

func FetchJSON(req FetchRequest, v interface{}) error {
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	if req.Async {
		return fmt.Errorf("Called synchronous FetchJSON with async request")
	}
	req.Headers["Accept"] = "application/json"
	req.Headers["Content-Type"] = "application/json"
	bits, err := FetchBody(req)
	if err != nil {
		return err
	}
	return json.Unmarshal(bits, v)
}
