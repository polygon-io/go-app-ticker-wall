package main

// ScreenClientSlice is sortable. fancy.
type ScreenClientSlice []*ScreenClient

func (a ScreenClientSlice) Len() int           { return len(a) }
func (a ScreenClientSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ScreenClientSlice) Less(i, j int) bool { return a[i].Screen.Index < a[j].Screen.Index }
