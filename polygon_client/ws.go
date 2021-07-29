package polygon

import (
	"context"
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

//easyjson:json
type websocketTrades []websocketTrade

//easyjson:json
type websocketTrade struct {
	Event string  `json:"ev"`
	ID    string  `json:"i"`
	Price float64 `json:"p"`
	// This is a hack. We use the 'high' price if it's an aggregate because trades
	// also have a 'c' attribute, with a different type, which causes havoc.
	High   float64 `json:"h"`
	Ticker string  `json:"sym"`
}

// AddTickerToUpdates sends a subscribe message to the given tickers.
func (c *Client) AddTickerToUpdates(tickers []string) error {
	// Determine the prefix character.
	prefix := "A."
	if c.perTickUpdates {
		prefix = "T."
	}

	// Create the correct subscription parameters. ( Trades vs Aggregates ).
	subItems := make([]string, 0, len(tickers))
	for _, ticker := range tickers {
		subItems = append(subItems, prefix+ticker)
	}

	// Subscribe to updates for these items.
	subscribeString := fmt.Sprintf(`{"action":"subscribe","params":"%s"}`, strings.Join(subItems, ","))
	if err := c.wsClient.WriteMessage(websocket.TextMessage, []byte(subscribeString)); err != nil {
		return fmt.Errorf("unable to send subscription message to websockets: %w", err)
	}

	return nil
}

// ListenForTickerUpdates listens for trades on the given tickers. This will
// get propogated via the client.PriceUpdates channel.
func (c *Client) ListenForTickerUpdates(ctx context.Context, tickers []string) error {
	// Close our update channel when we exit.
	defer close(c.tickerUpdate)

	logrus.Debug("tickerS: ", len(tickers), tickers)

	wsClient, _, err := websocket.DefaultDialer.DialContext(ctx, "wss://socket.polygon.io/stocks", nil)
	if err != nil {
		return fmt.Errorf("unable to connect to websocket endpoint: %w", err)
	}
	c.wsClient = wsClient
	defer c.wsClient.Close()

	// Set close handler.
	// TODO: Reconnect when websockets gets disconnected.
	c.wsClient.SetCloseHandler(func(code int, text string) error {
		logrus.Debug("WebSockets closed.")
		return nil
	})

	if err := c.wsClient.WriteMessage(
		websocket.TextMessage, []byte(fmt.Sprintf(`{"action":"auth","params":"%s"}`, c.APIKey)),
	); err != nil {
		return fmt.Errorf("unable to send auth message to websockets: %w", err)
	}

	if err := c.AddTickerToUpdates(tickers); err != nil {
		return err
	}

	// Start the actual processing.
	// Starting a go routing in a library func is not great. Also, completely
	// ignores the error returned.
	go func() {
		if err := c.queueTickerUpdates(ctx); err != nil {
			logrus.WithError(err).Error("Unable to queue ticker update.")
		}
	}()
	go func() {
		<-ctx.Done()
		logrus.Debug("Context closed, ending WS connection.")
		_ = c.wsClient.Close()
	}()

	// As little logic as possible in the reader loop:
	logrus.Debug("Listening to WebSockets for updates.")
	for {
		// Read message from WS.
		_, messageBody, err := c.wsClient.ReadMessage()
		if err != nil {
			return fmt.Errorf("websocket read error: %w", err)
		}

		// Send it to the parser.
		c.tickerUpdate <- messageBody
	}
}

func (c *Client) queueTickerUpdates(ctx context.Context) error {
	// Close our update channel when we exit.
	defer close(c.PriceUpdates)
	logrus.Debug("Queue ticker updates")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msgBytes, ok := <-c.tickerUpdate:
			if !ok {
				return nil
			}

			// Actually process
			if err := c.processWebsocketEvent(msgBytes); err != nil {
				return err
			}
		}
	}
}

func (c *Client) processWebsocketEvent(msgBytes []byte) error {
	// Unmarshal the WebSocket message.
	trades := websocketTrades{}
	if err := trades.UnmarshalJSON(msgBytes); err != nil {
		return fmt.Errorf("could not unmarshal json from server: %w", err)
	}

	// Each message contains multiple events inside of it.
	for _, trade := range trades {
		// This is NOT a trade message, skip.
		if trade.Event != "T" && trade.Event != "A" {
			continue
		}

		price := trade.Price
		if trade.Event == "A" {
			price = trade.High
		}

		// Broadcast this update.
		c.PriceUpdates <- &models.PriceUpdate{
			Ticker: trade.Ticker,
			Price:  price,
		}
	}

	return nil
}
