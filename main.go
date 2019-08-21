package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/360EntSecGroup-Skylar/excelize"
)

const port = 3885
const invalidDateFormat = "Invalid date format. Please use mm-dd"

// represents the raw json data received from a receiptbox client
type message struct {
	Date       string `json:"date"`
	Restaurant string `json:"restaurant"`
	Amount     string `json:"amount"`
}

func main() {

	http.HandleFunc("/", indexHandler)

	fmt.Printf("Running receiptbox_server v0.1 on port %d\n", port)

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	log.Fatal(err)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}
	fmt.Printf("> received %s request\n", r.Method)

	// read the sent data as bytes
	body, err := ioutil.ReadAll(r.Body)
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

	entry, err := createEntry(msg)
	if err != nil {
		fmt.Fprint(w, err.Error())
		return
	}

	fmt.Printf("Date: %s\n", entry.Date.String())
	fmt.Printf("Restaurant: %s\n", entry.Restaurant)
	fmt.Printf("Amount: %s\n", entry.Amount.StringFixedBank(2))

	updateSheet(entry)

	fmt.Fprint(w, "OK")
}

func updateSheet(e *entry) {

	f, err := excelize.OpenFile("./receipts.xlsx")
	if err != nil {
		log.Fatal(err)
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		log.Fatal(err)
	}

	// calculate the first empty row
	rowNumber := len(rows) + 1

	f.SetCellValue("Sheet1", fmt.Sprintf("A%d", rowNumber), e.Date.Format("January 02, 2006"))
	f.SetCellValue("Sheet1", fmt.Sprintf("B%d", rowNumber), e.Restaurant)
	f.SetCellValue("Sheet1", fmt.Sprintf("C%d", rowNumber), e.Amount.StringFixedBank(2))

	if err = f.SaveAs("./receipts.xlsx"); err != nil {
		log.Fatal(err)
	}
}
