package leader

import "github.com/polygon-io/go-app-ticker-wall/models"

// UpdateClientSlice is sortable. fancy.
type UpdateClientSlice []*UpdateClient

func (a UpdateClientSlice) Len() int           { return len(a) }
func (a UpdateClientSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a UpdateClientSlice) Less(i, j int) bool { return a[i].Screen.Index < a[j].Screen.Index }

// constructRGBA is a helper which turns our env variables ( map of string -> int32 ) into a struct.
func constructRGBA(colorMap map[string]int32) *models.RGBA {
	return &models.RGBA{
		Red:   colorMap["red"],
		Green: colorMap["green"],
		Blue:  colorMap["blue"],
		Alpha: colorMap["alpha"],
	}
}
