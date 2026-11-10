package holiday

// MessageResponse is the data payload returned by holiday and holiday-category
// delete endpoints.
type MessageResponse struct {
	Message string `json:"message"`
}
