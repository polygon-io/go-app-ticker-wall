package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

// We SERIOUSLY need our own Go library... wtf lol

type polygonAPIResponse struct {
	Ticker    string `json:"ticker"`
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
}

type polygonAggregate struct {
	Ticker    string  `json:"T"`
	Timestamp int64   `json:"t"`
	Volume    float64 `json:"v"`
	Close     float64 `json:"c"`
	Open      float64 `json:"o"`
	High      float64 `json:"h"`
	Low       float64 `json:"l"`
}

type polygonTrade struct {
	Price float64 `json:"p"`
}

type polygonPreviousClose struct {
	polygonAPIResponse
	Results []*polygonAggregate `json:"results"`
}

type polygonLastTrade struct {
	polygonAPIResponse
	Results polygonTrade `json:"results"`
}

func (t *TickerWallLeader) loadInitialTickerData(ctx context.Context, tickerSym string) (*models.Ticker, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	logrus.WithField("ticker", tickerSym).Debug("Loading initial ticker data..")
	ticker := &models.Ticker{
		Ticker:      tickerSym,
		CompanyName: tickerSym, // Here for now until we implement reference ticker details.
	}

	// Get Yesterdays Price
	previousClosePrice, err := GetTickerYesterdaysClose(ctx, t.cfg.APIKey, tickerSym)
	if err != nil {
		return nil, err
	}
	ticker.PreviousClosePrice = previousClosePrice

	// Get Current Price
	currentPrice, err := GetTickerCurrentPrice(ctx, t.cfg.APIKey, tickerSym)
	if err != nil {
		return nil, err
	}
	ticker.Price = currentPrice

	return ticker, nil
}

func GetTickerCurrentPrice(ctx context.Context, apiKey, ticker string) (float64, error) {
	url := "https://api.polygon.io/v2/last/trade/" + ticker + "?apiKey=" + apiKey
	body, err := makeHTTPRequest(ctx, url)
	if err != nil {
		return 0, err
	}

	res := &polygonLastTrade{}
	if err := json.Unmarshal(body, res); err != nil {
		return 0, fmt.Errorf("unable to parse JSON response from polygon: %w", err)
	}

	return res.Results.Price, nil
}

func GetTickerYesterdaysClose(ctx context.Context, apiKey, ticker string) (float64, error) {
	url := "https://api.polygon.io/v2/aggs/ticker/" + ticker + "/prev?apiKey=" + apiKey
	body, err := makeHTTPRequest(ctx, url)
	if err != nil {
		return 0, err
	}

	res := &polygonPreviousClose{}
	if err := json.Unmarshal(body, res); err != nil {
		return 0, fmt.Errorf("unable to parse JSON response from polygon: %w", err)
	}

	logrus.Debug("Parsed the previous clsoe: ", res)
	if len(res.Results) < 1 {
		return 0, fmt.Errorf("no previous close found: %w", err)
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
