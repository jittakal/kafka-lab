package events

import "time"

// BookIssuedData represents the data payload for a book issued event
type BookIssuedData struct {
	BookID      string    `json:"bookId"`
	Title       string    `json:"title"`
	ISBN        string    `json:"isbn"`
	Author      string    `json:"author"`
	Category    string    `json:"category"`
	MemberID    string    `json:"memberId"`
	MemberName  string    `json:"memberName"`
	MemberEmail string    `json:"memberEmail"`
	IssueDate   time.Time `json:"issueDate"`
	DueDate     time.Time `json:"dueDate"`
	LibraryID   string    `json:"libraryId"`
	BranchName  string    `json:"branchName"`
}

// BookReturnedData represents the data payload for a book returned event
type BookReturnedData struct {
	BookID        string    `json:"bookId"`
	Title         string    `json:"title"`
	ISBN          string    `json:"isbn"`
	MemberID      string    `json:"memberId"`
	MemberName    string    `json:"memberName"`
	IssueDate     time.Time `json:"issueDate"`
	ReturnDate    time.Time `json:"returnDate"`
	DueDate       time.Time `json:"dueDate"`
	IsLate        bool      `json:"isLate"`
	LateDays      int       `json:"lateDays"`
	LateFeeAmount float64   `json:"lateFeeAmount"`
	LibraryID     string    `json:"libraryId"`
	BranchName    string    `json:"branchName"`
	Condition     string    `json:"condition"` // good, fair, damaged
}

// Event type constants
const (
	EventTypeBookIssued   = "com.library.books.issued"
	EventTypeBookReturned = "com.library.books.returned"

	EventSource = "library-management-system"

	// CloudEvents content type
	ContentTypeJSON = "application/json"
)
