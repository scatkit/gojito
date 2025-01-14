package searcher_client

import (
	"net/http"
	"net/url"
)

type Encoding string

var (
	Base64 Encoding
	Base58 Encoding
)

var DefaultHeader = http.Header{
	"Content-Type": {"application/json"},
	"User-Agent":   {"jito-go :)"},
}

var jitoBundleURL = &url.URL{
	Scheme: "https",
	Host:   "mainnet.block-engine.jito.wtf",
	Path:   "/api/v1/bundles",
}



type GetTipAccountsResponse struct {
	Jsonrpc string   `json:"jsonrpc"`
	Result  []string `json:"result"`
	Id      int      `json:"id"`
}

type TransactionResponse struct {
	Jsonrpc  string `json:"jsonrpc"`
	Result   string `json:"result"`
	ID       int    `json:"id"`
	BundleID string
}
