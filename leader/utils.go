package leader

// UpdateClientSlice is sortable. fancy.
type UpdateClientSlice []*UpdateClient

func (a UpdateClientSlice) Len() int           { return len(a) }
func (a UpdateClientSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a UpdateClientSlice) Less(i, j int) bool { return a[i].Screen.Index < a[j].Screen.Index }
