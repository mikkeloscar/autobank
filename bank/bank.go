package bank

import "time"

// Bank defines a bank interface.
type Bank interface {
	// Statements returns a table of statements in the defined period.
	Statements(from, to time.Time) ([][]string, error)
}
