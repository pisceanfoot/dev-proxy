package review

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("expected Reviewer, got nil")
	}
	if r.inputReader == nil {
		t.Fatal("expected inputReader to be set")
	}
	if r.outputWriter == nil {
		t.Fatal("expected outputWriter to be set")
	}
}

func TestSetInput(t *testing.T) {
	r := New()
	input := strings.NewReader("y\n")
	reader := bufio.NewReader(input)
	r.SetInput(reader)

	if r.inputReader != reader {
		t.Fatal("expected inputReader to be updated")
	}
}

func TestReviewRequest(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantDecision Decision
	}{
		{"approve yes", "y\n", Approve},
		{"approve yes lowercase", "yes\n", Approve},
		{"approve Y", "Y\n", Approve},
		{"discard n", "n\n", Discard},
		{"discard no", "no\n", Discard},
		{"modify m", "m\n", Modify},
		{"modify modify", "modify\n", Modify},
		{"empty defaults to discard", "\n", Discard},
		{"whitespace defaults to discard", "   \n", Discard},
		{"unknown defaults to discard", "x\n", Discard},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			var output bytes.Buffer
			r.outputWriter = &output
			r.SetInput(bufio.NewReader(strings.NewReader(tt.input)))

			req := &Request{
				Method:  http.MethodGet,
				URL:     "http://example.com/test",
				Headers: http.Header{"Content-Type": []string{"application/json"}},
				Body:    []byte(`{"key":"value"}`),
			}
			w := httptest.NewRecorder()

			got := r.ReviewRequest(req, w)
			if got != tt.wantDecision {
				t.Fatalf("expected decision %q, got %q", tt.wantDecision, got)
			}

			outStr := output.String()
			if !strings.Contains(outStr, "REQUEST REVIEW") {
				t.Fatal("expected output to contain 'REQUEST REVIEW'")
			}
			if !strings.Contains(outStr, req.Method) {
				t.Fatalf("expected output to contain method %q", req.Method)
			}
		})
	}
}

func TestReviewResponse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantDecision Decision
	}{
		{"approve yes", "y\n", Approve},
		{"approve yes lowercase", "yes\n", Approve},
		{"discard n", "n\n", Discard},
		{"empty defaults to discard", "\n", Discard},
		{"unknown defaults to discard", "x\n", Discard},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			var output bytes.Buffer
			r.outputWriter = &output
			r.SetInput(bufio.NewReader(strings.NewReader(tt.input)))

			resp := &Response{
				StatusCode: 200,
				Headers:    http.Header{"Content-Type": []string{"application/json"}},
				Body:       []byte(`{"result":"ok"}`),
			}
			w := httptest.NewRecorder()

			got := r.ReviewResponse(resp, w)
			if got != tt.wantDecision {
				t.Fatalf("expected decision %q, got %q", tt.wantDecision, got)
			}

			outStr := output.String()
			if !strings.Contains(outStr, "RESPONSE REVIEW") {
				t.Fatal("expected output to contain 'RESPONSE REVIEW'")
			}
		})
	}
}

func TestSendRequest(t *testing.T) {
	t.Run("approved", func(t *testing.T) {
		r := New()
		r.SetInput(bufio.NewReader(strings.NewReader("y\n")))

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		approved, err := r.SendRequest(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !approved {
			t.Fatal("expected approved=true")
		}
	})

	t.Run("discarded", func(t *testing.T) {
		r := New()
		r.SetInput(bufio.NewReader(strings.NewReader("n\n")))

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		approved, err := r.SendRequest(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if approved {
			t.Fatal("expected approved=false")
		}
	})
}

func TestReadAnswer(t *testing.T) {
	tests := []struct {
		name string
		input string
		want string
	}{
		{"empty returns n", "\n", "n"},
		{"whitespace trimmed", "  yes  \n", "yes"},
		{"normal answer", "approve\n", "approve"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			r.SetInput(bufio.NewReader(strings.NewReader(tt.input)))
			got := r.readAnswer()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"long string", "hello world", 5, "hello..."},
		{"empty string", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCloneRequest(t *testing.T) {
	t.Run("with body", func(t *testing.T) {
		body := strings.NewReader(`{"key":"value"}`)
		req := httptest.NewRequest(http.MethodPost, "http://example.com/test", body)
		req.Header.Set("Content-Type", "application/json")

		cloned, bodyBytes, err := CloneRequest(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cloned == nil {
			t.Fatal("expected cloned request, got nil")
		}
		if string(bodyBytes) != `{"key":"value"}` {
			t.Fatalf("expected body bytes %q, got %q", `{"key":"value"}`, string(bodyBytes))
		}

		// Verify original request body is still readable
		origBody, _ := io.ReadAll(req.Body)
		if string(origBody) != `{"key":"value"}` {
			t.Fatalf("expected original body %q, got %q", `{"key":"value"}`, string(origBody))
		}
	})

	t.Run("nil body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)

		cloned, bodyBytes, err := CloneRequest(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cloned == nil {
			t.Fatal("expected cloned request, got nil")
		}
		if len(bodyBytes) != 0 {
			t.Fatalf("expected empty body bytes, got %q", string(bodyBytes))
		}
	})
}
