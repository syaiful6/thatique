package types

// Pagination info requested by users
type PaginationInfo struct {
	Limit, Offset int32
}

type HTTPError struct {
	Message string `json:"message"`
	Code    int    `json:"-"`
}

func (e *HTTPError) Error() string {
	return e.Message
}
