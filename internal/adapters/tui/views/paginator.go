package views

// Paginator provides pagination logic for list views
type Paginator struct {
	pageSize   int
	pageOffset int
	cursor     int
	totalItems int
}

// NewPaginator creates a new paginator with the given page size
func NewPaginator(pageSize int) *Paginator {
	if pageSize <= 0 {
		pageSize = 10
	}
	return &Paginator{
		pageSize: pageSize,
	}
}

// SetTotal sets the total number of items
func (p *Paginator) SetTotal(total int) {
	p.totalItems = total
	// Adjust cursor if it's now out of bounds
	if p.cursor >= total && total > 0 {
		p.cursor = total - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	p.ensureCursorInPage()
}

// Cursor returns the current cursor position (absolute index)
func (p *Paginator) Cursor() int {
	return p.cursor
}

// SetCursor sets the cursor position
func (p *Paginator) SetCursor(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos >= p.totalItems && p.totalItems > 0 {
		pos = p.totalItems - 1
	}
	p.cursor = pos
	p.ensureCursorInPage()
}

// CursorUp moves the cursor up by one
func (p *Paginator) CursorUp() bool {
	if p.cursor > 0 {
		p.cursor--
		p.ensureCursorInPage()
		return true
	}
	return false
}

// CursorDown moves the cursor down by one
func (p *Paginator) CursorDown() bool {
	if p.cursor < p.totalItems-1 {
		p.cursor++
		p.ensureCursorInPage()
		return true
	}
	return false
}

// PageOffset returns the current page offset
func (p *Paginator) PageOffset() int {
	return p.pageOffset
}

// VisibleRange returns the start and end indices for the current page
func (p *Paginator) VisibleRange() (start, end int) {
	start = p.pageOffset
	end = min(p.pageOffset+p.pageSize, p.totalItems)
	return
}

// CursorInPage returns the cursor position relative to the current page
func (p *Paginator) CursorInPage() int {
	return p.cursor - p.pageOffset
}

// TotalPages returns the total number of pages
func (p *Paginator) TotalPages() int {
	if p.totalItems == 0 {
		return 1
	}
	return (p.totalItems + p.pageSize - 1) / p.pageSize
}

// CurrentPage returns the current page number (1-based)
func (p *Paginator) CurrentPage() int {
	return p.pageOffset/p.pageSize + 1
}

// NextPage moves to the next page
func (p *Paginator) NextPage() bool {
	if p.pageOffset+p.pageSize < p.totalItems {
		p.pageOffset += p.pageSize
		p.cursor = p.pageOffset
		return true
	}
	return false
}

// PrevPage moves to the previous page
func (p *Paginator) PrevPage() bool {
	if p.pageOffset > 0 {
		p.pageOffset -= p.pageSize
		if p.pageOffset < 0 {
			p.pageOffset = 0
		}
		p.cursor = p.pageOffset
		return true
	}
	return false
}

// Reset resets the paginator to its initial state
func (p *Paginator) Reset() {
	p.cursor = 0
	p.pageOffset = 0
	p.totalItems = 0
}

// ensureCursorInPage ensures cursor is within the current page
func (p *Paginator) ensureCursorInPage() {
	if p.cursor < p.pageOffset {
		p.pageOffset = (p.cursor / p.pageSize) * p.pageSize
	} else if p.cursor >= p.pageOffset+p.pageSize {
		p.pageOffset = (p.cursor / p.pageSize) * p.pageSize
	}
}

// RemoveAtCursor removes the item at the current cursor position.
// It adjusts the cursor and total count appropriately.
// Returns the new cursor position.
func (p *Paginator) RemoveAtCursor() int {
	if p.totalItems == 0 {
		return 0
	}
	p.totalItems--
	if p.cursor >= p.totalItems && p.cursor > 0 {
		p.cursor = p.totalItems - 1
	}
	p.ensureCursorInPage()
	return p.cursor
}
