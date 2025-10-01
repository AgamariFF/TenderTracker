package urlgen

import (
	"net/url"
)

type URLEncoder struct {
	baseURL string
	params  url.Values
}

func NewURLEncoder(baseURL string) *URLEncoder {
	return &URLEncoder{
		baseURL: baseURL,
		params:  url.Values{},
	}
}

func (e *URLEncoder) AddParam(key, value string) *URLEncoder {
	e.params.Add(key, value)
	return e
}

func (e *URLEncoder) AddParams(params map[string]string) *URLEncoder {
	for key, value := range params {
		e.params.Add(key, value)
	}
	return e
}

func (e *URLEncoder) AddArrayParam(key string, values []string) *URLEncoder {
	for _, value := range values {
		e.params.Add(key, value)
	}
	return e
}

func (e *URLEncoder) Build() string {
	if len(e.params) == 0 {
		return e.baseURL
	}
	return e.baseURL + "?" + e.params.Encode()
}

func (e *URLEncoder) String() string {
	return e.Build()
}

func ReplaceURLParam(urlStr, param, value string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	query := u.Query()
	query.Set(param, value)
	u.RawQuery = query.Encode()

	return u.String()
}

const (
	DFO  = "OKER36"
	PFO  = "OKER33"
	SZFO = "OKER31"
	SKFO = "OKER38"
	SFO  = "OKER35"
	YFO  = "OKER34"
	CFO  = "OKER30"
	UFO  = "OKER37"
)
