package viewmanager

type VirtualGrid struct{}

func NewVirtualGrid() *VirtualGrid {
	return &VirtualGrid{}
}

// GetScreenPosition takes a logical document Position and translates it
// to a virtual screen coordinate, applying offsets (e.g. scroll offsets, margins)
func (vg *VirtualGrid) GetScreenPosition(pos Position) Position {
	// In the future, this can be expanded to support scroll offsets, line wrapping,
	// tab character expansions, or adding space for line numbers (e.g. Col + 4).
	return pos
}
