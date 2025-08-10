package api

import "testing"

// Tests parseFlexibleEventDate for various accepted formats.
func TestParseFlexibleEventDate(t *testing.T) {
    cases := []struct{ in string; allDay bool }{
        {"2025-08-10T12:30:00Z", false},
        {"2025-08-10T12:30:00.123Z", false},
        {"2025-08-10T12:30", false},
        {"2025-08-10 12:30", false},
        {"2025-08-10", true},
    }
    for _, c := range cases {
        if _, err := parseFlexibleEventDate(c.in, c.allDay); err != nil {
            t.Errorf("expected success for %s got %v", c.in, err)
        }
    }
    if _, err := parseFlexibleEventDate("2025/08/10", false); err == nil {
        t.Error("expected error for unsupported format")
    }
}
