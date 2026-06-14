package aigateway

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type RawSSEEvent struct {
	Event string `json:"event"`
	Data  string `json:"data"`
	Done  bool   `json:"done"`
}

func ParseSSE(r io.Reader) ([]RawSSEEvent, error) {
	events := []RawSSEEvent{}
	if err := ScanSSE(r, func(event RawSSEEvent) error {
		events = append(events, event)
		return nil
	}); err != nil {
		return nil, err
	}
	return events, nil
}

func ScanSSE(r io.Reader, emit func(RawSSEEvent) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var eventName string
	var dataLines []string
	flush := func() error {
		if len(dataLines) == 0 {
			eventName = ""
			return nil
		}
		data := strings.Join(dataLines, "\n")
		event := RawSSEEvent{
			Event: eventName,
			Data:  data,
			Done:  strings.TrimSpace(data) == "[DONE]",
		}
		eventName = ""
		dataLines = nil
		if err := emit(event); err != nil {
			return err
		}
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "event":
			eventName = value
		case "data":
			dataLines = append(dataLines, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan sse: %w", err)
	}
	if err := flush(); err != nil {
		return err
	}
	return nil
}
