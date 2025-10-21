package parserbidzaar

import (
	"net/http"
	"time"
)

type Parser struct {
	client *http.Client
}

func NewParser() *Parser {
	return &Parser{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}
