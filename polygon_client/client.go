package polygon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

// We SERIOUSLY need our own Go library... wtf lol
// This library is awful and is a stop gap until we have a client library.
// It also reaches across modules, and does other bad things.
// Shame... Shame... Shame...

// bufferedChannelSize defines how many items we buffer internally before we start blocking.
const bufferedChannelSize = 10_000

type Client struct {
	APIKey       string
	PriceUpdates chan *models.PriceUpdate
	// Used for internally passing messages between websockets and parser.
	tickerUpdate   chan []byte
	perTickUpdates bool
	wsClient       *websocket.Conn
}

// NewClient creates a new polygon API client.
func NewClient(apiKey string, perTickUpdate bool) *Client {
	return &Client{
		APIKey:         apiKey,
		tickerUpdate:   make(chan []byte, bufferedChannelSize),
		PriceUpdates:   make(chan *models.PriceUpdate, bufferedChannelSize),
		perTickUpdates: perTickUpdate,
	}
}

func (c *Client) LoadTickerData(ctx context.Context, tickerSymbol string) (*models.Ticker, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	logrus.WithField("ticker", tickerSymbol).Debug("Loading ticker data..")
	ticker := &models.Ticker{
		Ticker: tickerSymbol,
	}

	// Get Yesterdays Price
	previousClosePrice, err := GetTickerYesterdaysClose(ctx, c.APIKey, tickerSymbol)
	if err != nil {
		return nil, err
	}
	ticker.PreviousClosePrice = previousClosePrice

	// Get Current Price
	currentPrice, err := GetTickerCurrentPrice(ctx, c.APIKey, tickerSymbol)
	if err != nil {
		return nil, err
	}
	ticker.Price = currentPrice

	// Get Company Info
	companyInfo, err := GetCompanyDetails(ctx, c.APIKey, tickerSymbol)
	if err != nil {
		return nil, err
	}
	ticker.CompanyName = companyInfo.CompanyName
	ticker.OutstandingShares = companyInfo.OutstandingShares

	return ticker, nil
}

func GetTickerCurrentPrice(ctx context.Context, apiKey, ticker string) (float64, error) {
	url := "https://api.polygon.io/v2/last/trade/" + ticker + "?apiKey=" + apiKey
	body, err := makeHTTPRequest(ctx, url)
	if err != nil {
		return 0, err
	}

	res := &LastTrade{}
	if err := json.Unmarshal(body, res); err != nil {
		return 0, fmt.Errorf("unable to parse JSON response from polygon: %w", err)
	}

	return res.Results.Price, nil
}

func GetCompanyDetails(ctx context.Context, apiKey, ticker string) (*Company, error) {
	url := "https://api.polygon.io/vX/reference/tickers/" + ticker + "?apiKey=" + apiKey
	body, err := makeHTTPRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	res := &CompanyDetails{}
	if err := json.Unmarshal(body, res); err != nil {
		return nil, fmt.Errorf("unable to parse JSON response from polygon: %w", err)
	}

	return &res.Results, nil
}

func GetTickerYesterdaysClose(ctx context.Context, apiKey, ticker string) (float64, error) {
	url := "https://api.polygon.io/v2/aggs/ticker/" + ticker + "/prev?apiKey=" + apiKey
	body, err := makeHTTPRequest(ctx, url)
	if err != nil {
		return 0, err
	}

	res := &PreviousClose{}
	if err := json.Unmarshal(body, res); err != nil {
		return 0, fmt.Errorf("unable to parse JSON response from polygon: %w", err)
	}

	logrus.Debug("Parsed the previous clsoe: ", res)
	if len(res.Results) < 1 {
		return 0, nil
	}

	return res.Results[0].Close, nil
}

func makeHTTPRequest(ctx context.Context, url string) ([]byte, error) {
	logrus.WithField("url", url).Debug("Making API Request")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to make HTTP request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable read response body contents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("received non 200 response code")
	}

	return io.ReadAll(resp.Body)
}
