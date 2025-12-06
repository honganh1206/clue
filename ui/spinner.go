package ui

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

var (
	SpinnerBinary = []string{
		"010010",
		"001100",
		"100101",
		"111010",
		"111101",
		"010111",
		"101011",
		"111000",
		"110011",
		"110101",
	}

	SpinnerDots = []string{
		"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
	}

	SpinnerStar = []string{
		"+",
		"x",
		"*",
	}

	SpinnerLines = []string{
		"|", "/", "-", "\\",
	}
)

type Spinner struct {
	message      atomic.Value
	messageWidth int
	parts        []string
	value        int
	ticker       *time.Ticker
	started      time.Time
	stopped      time.Time
}

func NewSpinner(message string, parts []string) *Spinner {
	if len(parts) == 0 {
		parts = SpinnerBinary
	}
	s := &Spinner{
		parts:   parts,
		started: time.Now(),
	}
	s.SetMessage(message)
	go s.start()
	return s
}

func (s *Spinner) SetMessage(message string) {
	s.message.Store(message)
}

// Display the spinner with a message
func (s *Spinner) String() string {
	var sb strings.Builder

	if s.stopped.IsZero() {
		spinner := s.parts[s.value]
		sb.WriteString(spinner)
		sb.WriteString(" ")
	}

	if message, ok := s.message.Load().(string); ok && len(message) > 0 {
		message := strings.TrimSpace(message)
		if s.messageWidth > 0 && len(message) > s.messageWidth {
			// Prevent the message from wrapping to new lines or overflowing the display area
			message = message[:s.messageWidth]
		}

		// Write to string builder
		fmt.Fprintf(&sb, "%s", message)
		if padding := s.messageWidth - sb.Len(); padding > 0 {
			// Pad the message with space to reach the full messageWidth
			// to ensure consistent spacing between the message and spinner.
			sb.WriteString(strings.Repeat(" ", padding))
		}

		sb.WriteString(" ")
	}

	return sb.String()
}

func (s *Spinner) start() {
	s.ticker = time.NewTicker(100 * time.Millisecond)
	// Ticks are delivered via channel C
	for range s.ticker.C {
		// Use modulo to wrap around i.e., change the s.value to indices of the parts array
		s.value = (s.value + 1) % len(s.parts)
		if !s.stopped.IsZero() {
			return
		}
	}
}

func (s *Spinner) Stop() {
	if s.stopped.IsZero() {
		s.stopped = time.Now()
	}
}
