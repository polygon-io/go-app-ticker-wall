package manager

import (
	"math"
	"sort"
	"sync"

	"github.com/sirupsen/logrus"
)

// TickerManager is the interface used for ticker managers.
type TickerManager interface {
	// AddTicker adds a ticker symbol to this manager. It will be added at the correct index ( sorted alphabetically )
	AddTicker(ticker string, price float64, priceChangePercentage float64, companyName string)

	// DetermineTickersForRender takes a global offset and returns the ticker indices which should be rendered.
	DetermineTickersForRender(globalOffset int) (tickers []*Ticker)

	// TickerOffset determines what the offset should be for this ticker, on this screen.
	TickerOffset(globalOffset int, ticker *Ticker) int
}

type PresentationData struct {
	ScreenGlobalOffset int // ScreenGlobalOffset is the offset of this screen in the entire network. Eg: If there are 2 monitors before this one, each 500px. Our screen global offset will be 1000px.
	ScreenWidth        int // ScreenWidth is the width of the entire applications screen ( px ).
	ScreenHeight       int // ScreenHeight is the height of the entire applications screen ( px ).
	TickerBoxWidth     int // TickerBoxWidth is the total px width we allocate per ticker.
	ScreenIndex        int // ScreenIndex is index of this screen in the sequence of displays. ( left to right ) ( starting at 1 ).
}

// DefaultManager is used to manage all tickers for an application. It knows important information for rendering.
type DefaultManager struct {
	sync.RWMutex

	// Constant presentation data:
	PresentationData *PresentationData

	// Our list of tickers.
	Tickers []*Ticker // Tickers is the entire list of tickers. We use a map for faster updates.
}

// NewDefaultManager creates a new default ticker manager.
func NewDefaultManager(presentationData *PresentationData) TickerManager {
	mgr := &DefaultManager{
		PresentationData: presentationData,
	}
	return mgr
}

func (m *DefaultManager) TickerOffset(globalOffset int, ticker *Ticker) int {
	localizedOffset := (globalOffset % (len(m.Tickers) * m.PresentationData.TickerBoxWidth))

	offset := ((ticker.Index * m.PresentationData.TickerBoxWidth) - localizedOffset) - m.PresentationData.ScreenGlobalOffset
	// Too far left, probably need to wrap it around.
	if offset < 0 {
		if offset < -(m.PresentationData.TickerBoxWidth) {
			offset = (len(m.Tickers) * m.PresentationData.TickerBoxWidth) - int(math.Abs(float64(offset)))
		}
	}
	return offset
}

func (m *DefaultManager) DetermineTickersForRender(globalOffset int) []*Ticker {
	var tickers []*Ticker

	// Global offset does not necessarily ever reset, so we need to get the localized offset.
	localizedOffset := (globalOffset % (len(m.Tickers) * m.PresentationData.TickerBoxWidth)) + m.PresentationData.ScreenGlobalOffset
	logrus.Trace("Localized Offset: ", localizedOffset)

	firstIndex := int(math.Floor(float64(localizedOffset) / float64(m.PresentationData.TickerBoxWidth)))
	lastIndex := int(math.Floor(float64(localizedOffset+m.PresentationData.ScreenWidth) / float64(m.PresentationData.TickerBoxWidth)))

	logrus.Trace("offsets: ", firstIndex, lastIndex)

	// eg: -2
	if firstIndex < 0 {
		boundedFirst := int(float64(len(m.Tickers)) - math.Abs(float64(firstIndex)))
		logrus.Trace("first index short: ", boundedFirst)
		tickers = append(tickers, m.Tickers[boundedFirst:]...)
		// Now we set first index to 0 since we have the overflow items.
		firstIndex = 0
	}

	// If our end index is outside of the bounds.
	boundedLastIndex := lastIndex
	if lastIndex+1 > len(m.Tickers) {
		boundedLastIndex = len(m.Tickers) - 1
		logrus.Trace("last index long: ", boundedLastIndex)
	}

	// Add our valid section.
	tickers = append(tickers, m.Tickers[firstIndex:boundedLastIndex+1]...)

	// If we have overflow, now add those.
	if lastIndex+1 > len(m.Tickers) {
		boundedLast := lastIndex - len(m.Tickers)
		tickers = append(tickers, m.Tickers[:boundedLast+1]...)
		logrus.Trace("last index long2: ", boundedLast)
	}

	return tickers
}

// AddTicker creates a new ticker in this manager. If the ticker exists, it will be replaced. Tickers are unique by their ticker symbol.
func (m *DefaultManager) AddTicker(ticker string, price float64, priceChangePercentage float64, companyName string) {
	m.Lock()
	defer m.Unlock()

	// Create a new ticker.
	tickerObj := &Ticker{
		Ticker:                ticker,
		Price:                 price,
		PriceChangePercentage: priceChangePercentage,
		CompanyName:           companyName,
	}

	// Add ticker to our list.
	m.Tickers = append(m.Tickers, tickerObj)

	// Sort our tickers now we have a new one.
	sort.Sort(TickerSlice(m.Tickers))

	// Update everyones index, since this could potentially change it.
	for i, ticker := range m.Tickers {
		ticker.Index = i
	}
}
