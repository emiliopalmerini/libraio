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

// MessageHandler is an interface for views that can handle their own messages.
// This enables the Open/Closed principle - new views can be added without
// modifying the app's main Update switch statement.
type MessageHandler interface {
	// HandleMessage processes a message and returns any commands to execute.
	// Returns (handled, cmd) where handled is true if the message was processed.
	HandleMessage(msg any) (handled bool, result MessageResult)
}

// MessageResult contains the outcome of handling a message
type MessageResult struct {
	// SwitchTo indicates a view to switch to (empty string means stay)
	SwitchTo ViewID
	// ReloadBrowser indicates the browser should reload its tree
	ReloadBrowser bool
	// Message is an optional message to display
	Message string
	// IsError indicates if Message is an error
	IsError bool
}

// ViewRouter manages routing between views.
// It provides a centralized way to handle view switching and message routing.
type ViewRouter struct {
	handlers map[ViewID]MessageHandler
}

// NewViewRouter creates a new view router
func NewViewRouter() *ViewRouter {
	return &ViewRouter{
		handlers: make(map[ViewID]MessageHandler),
	}
}

// Register registers a message handler for a view
func (r *ViewRouter) Register(id ViewID, handler MessageHandler) {
	r.handlers[id] = handler
}

// Route attempts to route a message to the appropriate handler.
// Returns (handled, result) where handled is true if any handler processed the message.
func (r *ViewRouter) Route(msg any) (bool, MessageResult) {
	// Try each registered handler
	for _, handler := range r.handlers {
		if handled, result := handler.HandleMessage(msg); handled {
			return true, result
		}
	}
	return false, MessageResult{}
}
