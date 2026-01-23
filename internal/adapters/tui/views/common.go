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

// ViewID identifies which view an operation originated from
type ViewID string

const (
	ViewIDCreate       ViewID = "create"
	ViewIDMove         ViewID = "move"
	ViewIDDelete       ViewID = "delete"
	ViewIDArchive      ViewID = "archive"
	ViewIDSmartCatalog ViewID = "smartcatalog"
)

// OperationSuccessMsg is a generic success message that can be used by any view.
// ViewID identifies which operation succeeded for routing in app.go.
type OperationSuccessMsg struct {
	ViewID  ViewID
	Message string
}

// OperationErrMsg is a generic error message that can be used by any view.
// ViewID identifies which operation failed for routing in app.go.
type OperationErrMsg struct {
	ViewID ViewID
	Err    error
}
