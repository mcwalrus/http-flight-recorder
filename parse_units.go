package flightrecorder

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// UnmarshalJSON unmarshals the status response payload.
// It supports both Go duration and memory unit formats.
func (s *StatusResponse) MarshalJSON() ([]byte, error) {
	type Alias struct {
		Enabled bool   `json:"enabled"`
		Period  string `json:"period"`
		Size    string `json:"size"`
	}
	var t Alias
	t.Enabled = s.Enabled
	t.Period = s.Period.String()
	if s.Size != 0 {
		t.Size = formatMemoryUnits(s.Size)
	} else {
		t.Size = "0B"
	}
	return json.Marshal(t)
}

// UnmarshalJSON unmarshals the update request payload.
// It supports both Go duration and memory unit formats.
func (u *UpdateRequest) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Period *string `json:"period,omitempty"`
		Size   *string `json:"size,omitempty"`
	}
	var t Alias
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	u.Period = nil
	if t.Period != nil {
		duration, err := time.ParseDuration(*t.Period)
		if err != nil {
			return fmt.Errorf("invalid period: %s should be a duration (e.g. 1s, 100ms, 1h)", *t.Period)
		}
		u.Period = &duration
	}
	if t.Size != nil {
		size, err := parseUnitsBytes(*t.Size)
		if err != nil {
			return fmt.Errorf("invalid size: %s should be an integer of bytes, or a memory unit (e.g. X, or 1MB, 1KB, 1B)", *t.Size)
		}
		u.Size = &size
	}
	return nil
}

func formatMemoryUnits(s int) string {
	if s > 1024*1024 {
		return fmt.Sprintf("%dMB", s/(1024*1024))
	} else if s > 1024 {
		return fmt.Sprintf("%dKB", s/1024)
	} else {
		return fmt.Sprintf("%dB", s)
	}
}

func parseUnitsBytes(s string) (int, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "MB") {
		s = strings.TrimSuffix(s, "MB")
		return convertMemoryUnits(s, 1024*1024)
	} else if strings.HasSuffix(s, "KB") {
		s = strings.TrimSuffix(s, "KB")
		return convertMemoryUnits(s, 1024)
	} else if strings.HasSuffix(s, "B") {
		s = strings.TrimSuffix(s, "B")
	}
	return strconv.Atoi(s)
}

func convertMemoryUnits(s string, mult int) (int, error) {
	if v, err := strconv.Atoi(s); err != nil {
		return 0, err
	} else {
		return v * mult, nil
	}
}
