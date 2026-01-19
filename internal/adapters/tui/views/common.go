package views

// ViewState contains common state shared by all view models.
// Embed this struct in view models to get width/height and message handling.
type ViewState struct {
	Width      int
	Height     int
	Message    string
	MessageErr bool
}

// SetSize updates the view dimensions
func (s *ViewState) SetSize(width, height int) {
	s.Width = width
	s.Height = height
}

// SetMessage sets a message to display in the view
func (s *ViewState) SetMessage(msg string, isErr bool) {
	s.Message = msg
	s.MessageErr = isErr
}

// ClearMessage clears the current message
func (s *ViewState) ClearMessage() {
	s.Message = ""
	s.MessageErr = false
}
