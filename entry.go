package main

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// represents an actual entry with the right data types
type entry struct {
	Date       time.Time
	Restaurant string
	Amount     decimal.Decimal
}

func createEntry(msg message, year int) (*entry, error) {
	result := entry{}

	// get restaurant name
	result.Restaurant = msg.Restaurant

	// get cash amount
	amount, err := decimal.NewFromString(msg.Amount)
	if err != nil {
		return nil, errors.New("Cannot read Amount")
	}
	result.Amount = amount

	// get date
	dateComponents := strings.Split(msg.Date, "-")
	if len(dateComponents) != 2 {
		return nil, errors.New(invalidDateFormat)
	}
	month, err := strconv.Atoi(dateComponents[0])
	if err != nil {
		return nil, errors.New(invalidDateFormat)
	}
	if month > 12 || month < 1 {
		return nil, errors.New(invalidDateFormat)
	}
	day, err := strconv.Atoi(dateComponents[1])
	if err != nil {
		return nil, errors.New(invalidDateFormat)
	}
	if day > 31 || day < 1 {
		return nil, errors.New(invalidDateFormat)
	}
	result.Date = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// done
	return &result, nil
}
