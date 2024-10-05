package utoml

import (
	"encoding"
	"fmt"
	"strings"
	"time"

	"uw/utoml/unstable"
)

// LocalDate represents a calendar day in no specific timezone.
type LocalDate struct {
	Year  int
	Month int
	Day   int
}

var (
	_ = encoding.TextMarshaler(&LocalDate{})
	_ = encoding.TextUnmarshaler(&LocalDate{})
)

func (d *LocalDate) IsZero() bool {
	return d == nil || (d.Year <= 1970 && d.Month <= 1 && d.Day <= 1)
}

// AsTime converts d into a specific time instance at midnight in zone.
func (d *LocalDate) AsTime(zone *time.Location) time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, zone)
}

// String returns RFC 3339 representation of d.
func (d *LocalDate) String() string {
	if d.Year < 1970 {
		d.Year = 1970
	}
	if d.Month < 1 || d.Month > 12 {
		d.Month = 1
	}
	if d.Day < 1 || d.Day > daysIn(d.Month, d.Year) {
		d.Day = 1
	}

	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// MarshalText returns RFC 3339 representation of d.
func (d *LocalDate) MarshalText() ([]byte, error) {
	if d.IsZero() {
		return nil, nil
	}

	return []byte(d.String()), nil
}

// UnmarshalText parses b using RFC 3339 to fill d.
func (d *LocalDate) UnmarshalText(b []byte) error {
	if len(b) < 1 {
		return nil
	}

	res, err := parseLocalDate(b)
	if err != nil {
		return err
	}
	*d = res
	return nil
}

// LocalTime represents a time of day of no specific day in no specific
// timezone.
type LocalTime struct {
	Hour       int // Hour of the day: [0; 24[
	Minute     int // Minute of the hour: [0; 60[
	Second     int // Second of the minute: [0; 60[
	Nanosecond int // Nanoseconds within the second:  [0, 1000000000[
	Precision  int // Number of digits to display for Nanosecond.
}

var (
	_ = encoding.TextMarshaler(&LocalTime{})
	_ = encoding.TextUnmarshaler(&LocalTime{})
)

func (d *LocalTime) IsZero() bool {
	return d == nil || d.Hour == 0 || d.Minute == 0 || d.Second == 0
}

// String returns RFC 3339 representation of d.
// If d.Nanosecond and d.Precision are zero, the time won't have a nanosecond
// component. If d.Nanosecond > 0 but d.Precision = 0, then the minimum number
// of digits for nanoseconds is provided.
func (d *LocalTime) String() string {
	s := fmt.Sprintf("%02d:%02d:%02d", d.Hour, d.Minute, d.Second)

	if d.Precision > 0 {
		s += fmt.Sprintf(".%09d", d.Nanosecond)[:d.Precision+1]
	} else if d.Nanosecond > 0 {
		// Nanoseconds are specified, but precision is not provided. Use the
		// minimum.
		s += strings.Trim(fmt.Sprintf(".%09d", d.Nanosecond), "0")
	}

	return s
}

// MarshalText returns RFC 3339 representation of d.
func (d *LocalTime) MarshalText() ([]byte, error) {
	if d.IsZero() {
		return nil, nil
	}

	return []byte(d.String()), nil
}

// UnmarshalText parses b using RFC 3339 to fill d.
func (d *LocalTime) UnmarshalText(b []byte) error {
	if len(b) < 1 {
		return nil
	}

	res, left, err := parseLocalTime(b)
	if err == nil && len(left) != 0 {
		err = unstable.NewParserError(left, "extra characters")
	}
	if err != nil {
		return err
	}
	*d = res
	return nil
}

// LocalDateTime represents a time of a specific day in no specific timezone.
type LocalDateTime struct {
	LocalDate
	LocalTime
}

var (
	_ = encoding.TextMarshaler(&LocalDateTime{})
	_ = encoding.TextUnmarshaler(&LocalDateTime{})
)

func (d *LocalDateTime) IsZero() bool {
	return d == nil || d.LocalDate.IsZero() || d.LocalTime.IsZero()
}

// AsTime converts d into a specific time instance in zone.
func (d *LocalDateTime) AsTime(zone *time.Location) time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, d.Hour, d.Minute, d.Second, d.Nanosecond, zone)
}

// String returns RFC 3339 representation of d.
func (d *LocalDateTime) String() string {
	if d.IsZero() {
		return ""
	}

	return d.LocalDate.String() + "T" + d.LocalTime.String()
}

// MarshalText returns RFC 3339 representation of d.
func (d *LocalDateTime) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalText parses b using RFC 3339 to fill d.
func (d *LocalDateTime) UnmarshalText(data []byte) error {
	if len(data) < 1 {
		return nil
	}

	res, left, err := parseLocalDateTime(data)
	if err == nil && len(left) != 0 {
		err = unstable.NewParserError(left, "extra characters")
	}
	if err != nil {
		return err
	}

	*d = res
	return nil
}
