package leader

import "github.com/massive-com/go-app-ticker-wall/v2/models"

// Config handles the default settings, as well as data client auth.
type Config struct {
	TickerList string
	APIKey     string

	// Presentation Default Settings
	Presentation *models.PresentationSettings
}
