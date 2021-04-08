package manager

import (
	"sync"

	"github.com/polygon-io/go-app-ticker-wall/models"
)

type Ticker struct {
	sync.RWMutex
	models.Ticker
}
