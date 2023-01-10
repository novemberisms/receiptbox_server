package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
)

const port = 3885
const invalidDateFormat = "Invalid date format. Please use mm-dd"
const outputfile = "receipts.xlsx"

var mutex = sync.Mutex{}

var months = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}

// represents the raw json data received from a receiptbox client
type message struct {
	Date       string `json:"date"`
	Restaurant string `json:"restaurant"`
	Amount     string `json:"amount"`
}

var currentYear = time.Now().Year()

var currentTotal = 0.0

func main() {

	if len(os.Args) > 1 {
		var err error

		currentYear, err = strconv.Atoi(os.Args[1])

		if err != nil {
			log.Fatal(fmt.Sprintf("Cannot convert year '%s' to a number\n", os.Args[1]))
		}
	}

	http.HandleFunc("/", indexHandler)

	fmt.Printf("Running receiptbox_server on port %d\n", port)

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	log.Fatal(err)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}
	fmt.Printf("> received %s request\n", r.Method)

	// read the sent data as bytes
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	// decode the data into a message struct
	var msg message
	err = json.Unmarshal(body, &msg)
	if err != nil {
		log.Fatal(err)
	}

	entry, err := createEntry(msg, currentYear)
	if err != nil {
		fmt.Fprint(w, err.Error())
		return
	}

	fmt.Printf("Date: %s\n", entry.Date.String())
	fmt.Printf("Restaurant: %s\n", entry.Restaurant)
	fmt.Printf("Amount: %s\n", entry.Amount.StringFixedBank(2))

	setupSheet(outputfile)
	total := updateSheet(outputfile, entry)

	fmt.Fprintf(w, "OK: %s", strconv.FormatFloat(total, 'f', 2, 64))
}

func updateSheet(filename string, e *entry) float64 {

	fmt.Print("Updating sheet...")

	mutex.Lock()
	defer mutex.Unlock()

	f, err := excelize.OpenFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	sheetName := getSheetNameForDate(e.Date)
	rowNumber := findFirstEmptyRow(f, sheetName)

	f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNumber), e.Date.Format("January 02, 2006"))
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNumber), e.Restaurant)

	// convert the amount to a string and then back to a number to prevent excel format problems
	fixedAmountString := e.Amount.StringFixedBank(2)
	amount, _ := strconv.ParseFloat(fixedAmountString, 64)
	f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNumber), amount)

	// force the sheet to update formulas
	f.UpdateLinkedValue()

	if err = f.Save(); err != nil {
		log.Fatal(err)
	}

	currentTotal += e.Amount.InexactFloat64()

	fmt.Printf("OK. TOTAL: %.2f\n", currentTotal)

	return currentTotal
}

func setupSheet(filename string) {

	f, err := excelize.OpenFile(filename)
	if err != nil {
		fmt.Print("Creating new sheet...")
		// if the sheet does not exist yet, then create one
		f = excelize.NewFile()
		f.SaveAs(filename)
		fmt.Println("OK")
		setupSheet(filename)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	// determine if the sheet is already set up
	if f.GetSheetIndex(months[0]) != 0 {
		// if so, then if the current total is 0 then compute it
		if currentTotal == 0 {
			currentTotal = computeCurrentTotal(f)
		}
		return
	}

	fmt.Print("Setting up sheet...")

	for _, monthName := range months {
		f.NewSheet(monthName)
		f.SetColWidth(monthName, "A", "A", 20)
		f.SetColWidth(monthName, "B", "B", 32)
		f.SetColWidth(monthName, "C", "C", 10)
	}
	f.Save()

	fmt.Println("OK")
}

func getSheetNameForDate(date time.Time) string {
	for i, name := range months {
		if int(date.Month()) == i+1 {
			return name
		}
	}

	return "Sheet1"
}

func findFirstEmptyRow(f *excelize.File, sheetName string) int {
	rows := f.GetRows(sheetName)

	// calculate the first empty row
	for i, row := range rows {
		if len(row) == 0 {
			return i + 1
		}
		if strings.TrimSpace(row[0]) == "" {
			return i + 1
		}
	}

	// if control reaches here, then all previous rows have content and so we need to make a new one
	return len(rows) + 1
}

func computeCurrentTotal(f *excelize.File) float64 {

	total := 0.0

	fmt.Println("Computing total so far...")

	for _, month := range months {
		rows := f.GetRows(month)
		for _, row := range rows {
			if len(row) == 0 {
				continue
			}
			amountStr := row[2]
			if amount, err := strconv.ParseFloat(amountStr, 64); err == nil {
				total += amount
			}
		}
	}

	return total
}
