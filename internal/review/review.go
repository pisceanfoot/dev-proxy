package review

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Decision represents the reviewer's choice for a request.
type Decision string

const (
	Approve Decision = "approve"
	Discard Decision = "discard"
	Modify  Decision = "modify"
)

// Request captures an HTTP request for review.
type Request struct {
	Method  string
	URL     string
	Headers http.Header
	Body    []byte
}

// Response captures the upstream response for review.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// Reviewer handles interactive review of requests/responses.
type Reviewer struct {
	inputReader  *bufio.Reader
	outputWriter io.Writer
}

// New creates a Reviewer that reads from stdin and writes to stdout.
func New() *Reviewer {
	return &Reviewer{
		inputReader:  bufio.NewReader(nil),
		outputWriter: io.Discard,
	}
}

// SetInput sets the input reader (for testing or custom input sources).
func (r *Reviewer) SetInput(reader *bufio.Reader) {
	r.inputReader = reader
}

// ReviewRequest pauses and asks the user to approve/discard/modify a request.
func (r *Reviewer) ReviewRequest(req *Request, w http.ResponseWriter) Decision {
	fmt.Fprintf(r.outputWriter, "\n=== REQUEST REVIEW ===\n")
	fmt.Fprintf(r.outputWriter, "Method: %s\n", req.Method)
	fmt.Fprintf(r.outputWriter, "URL:    %s\n", req.URL)
	fmt.Fprintf(r.outputWriter, "Headers:\n")
	for k, v := range req.Headers {
		fmt.Fprintf(r.outputWriter, "  %s: %s\n", k, strings.Join(v, ", "))
	}
	if len(req.Body) > 0 {
		fmt.Fprintf(r.outputWriter, "Body (%d bytes): %s\n", len(req.Body), truncate(string(req.Body), 200))
	}
	fmt.Fprintf(r.outputWriter, "\n[y] Approve  [n] Discard  [m] Modify  ")

	answer := r.readAnswer()
	switch strings.ToLower(answer) {
	case "y", "yes":
		return Approve
	case "m", "modify":
		return Modify
	default:
		return Discard
	}
}

// ReviewResponse pauses and asks the user to approve/discard a response.
func (r *Reviewer) ReviewResponse(resp *Response, w http.ResponseWriter) Decision {
	fmt.Fprintf(r.outputWriter, "\n=== RESPONSE REVIEW ===\n")
	fmt.Fprintf(r.outputWriter, "Status: %d\n", resp.StatusCode)
	fmt.Fprintf(r.outputWriter, "Headers:\n")
	for k, v := range resp.Headers {
		fmt.Fprintf(r.outputWriter, "  %s: %s\n", k, strings.Join(v, ", "))
	}
	if len(resp.Body) > 0 {
		fmt.Fprintf(r.outputWriter, "Body (%d bytes): %s\n", len(resp.Body), truncate(string(resp.Body), 200))
	}
	fmt.Fprintf(r.outputWriter, "\n[y] Approve  [n] Discard  ")

	answer := r.readAnswer()
	switch strings.ToLower(answer) {
	case "y", "yes":
		return Approve
	default:
		return Discard
	}
}

// SendRequest sends a request to the review channel and waits for approval.
// Returns true if approved, false if discarded.
func (r *Reviewer) SendRequest(req *http.Request) (bool, error) {
	bodyBytes, _ := io.ReadAll(req.Body)
	defer req.Body.Close()

	reviewReq := &Request{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: req.Header.Clone(),
		Body:    bodyBytes,
	}

	fmt.Printf("\n[REVIEW] %s %s\n", reviewReq.Method, reviewReq.URL)
	fmt.Printf("Approve (y) or discard (n)? ")

	answer := r.readAnswer()
	return strings.ToLower(answer) == "y" || strings.ToLower(answer) == "yes", nil
}

func (r *Reviewer) readAnswer() string {
	line, _ := r.inputReader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return "n"
	}
	return line
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// CloneRequest creates a deep copy of an HTTP request for review.
func CloneRequest(req *http.Request) (*http.Request, []byte, error) {
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	cloned := req.Clone(req.Context())
	return cloned, bodyBytes, nil
}
