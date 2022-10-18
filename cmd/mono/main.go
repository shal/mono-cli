package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shal/mono"
	"github.com/shopspring/decimal"
)

const DateLayout = "2006-01-02"

type DateFlag struct {
	date time.Time
}

func (f *DateFlag) String() string {
	return f.date.Format(DateLayout)
}

func (f *DateFlag) Set(value string) error {
	date, err := time.Parse(DateLayout, value)
	if err != nil {
		return err
	}

	f.date = date

	return nil
}

type MonoWrap struct {
	mono *mono.Personal
}

func New(token string) *MonoWrap {
	client := mono.NewPersonal(token)
	return &MonoWrap{
		mono: client,
	}
}

func mccToENG(mcc int32) string {
	switch mcc {
	case 7841:
		return "VIDEO TAPE RENTAL STORES"
	case 4121:
		return "TAXICABS OR LIMOUSINES"
	case 5411:
		return "GROCERY STORES OR SUPERMARKETS"
	case 5812:
		return "RESTAURANTS OR EATING PLACES"
	case 5814:
		return "FAST FOOD RESTAURANTS"
	case 5912:
		return "PHARMCIES OR DRUG STORES"
	case 4829:
		return "MONEY ORDERS -- WIRE TRANSFER"
	case 7399:
		return "BUSINESS SERVICES (NEC)"
	case 5941:
		return "SPORTING GOODS STORES"
	case 5946:
		return "CAMERA AND/OR PHOTOGRAPHIC SUPPLY STORES"
	case 5651:
		return "FAMILY CLOTHING STORES"
	case 5541:
		return "GAS/SERVICE STATIONS WITH/WITHOUT ANCILLARY SERVICES"
	case 4812:
		return "TELECOMMUNCATIONS EQUIPMENT INCLUDING TELEPHONE SALES"
	case 4814:
		return "TELECOMMUNCATIONS SERV INCL LOCA/LONG DIST CREDIT & FAX"
	}

	return "UNKNOWN"
}

func mccToCategory(tx *mono.Transaction) string {
	description := strings.ToLower(tx.Description)

	switch tx.MCC {
	case 7841:
		return "Fun"
	case 4121:
		return "Transport/Taxi"
	case 5411:
		return "Food"
	case 5812, 5814, 5399:
		if strings.Contains(description, "coffee") {
			return "Coffee"
		}

		return "Restaurants"
	case 5912:
		return "Selfcare"
	case 7399:
		return "OpenCars"
	case 5941, 5946, 5651, 5699, 5732:
		return "Clothes"
	case 5541:
		return "Car/Gas"
	case 4812, 4814:
		return "Utility bills/Mobile"
	case 4829:
		if strings.Contains(description, "мама") {
			return "Mom/Father"
		}

		if strings.Contains(description, "megogo") {
			return "Зарплата"
		}

		if strings.Contains(description, "поповнення") {
			return "Charity"
		}

		return "Other"
	case 7538, 5533:
		return "Car/Fixes"
	case 5995:
		return "Cat"
	case 2741:
		return "Subscriptions"
	}

	return "Other"
}

func (w *MonoWrap) FindAccount(ctx context.Context, typ string) (*mono.Account, error) {
	user, err := w.mono.User(ctx)
	if err != nil {
		return nil, err
	}

	for i, acc := range user.Accounts {
		if string(acc.Type) == typ {
			ccy, err := mono.CurrencyFromISO4217(acc.CurrencyCode)
			if err != nil {
				continue
			}

			if ccy.Code == "UAH" {
				tmp := user.Accounts[i]
				return &tmp, nil
			}
		}
	}

	return nil, errors.New("not found")
}

func main() {
	to := DateFlag{date: time.Now()}
	from := DateFlag{to.date.Add(-24 * 31 * time.Hour)}

	var output string
	var account string

	flag.Var(&from, "from", "Start time")
	flag.Var(&to, "to", "Finish time")
	flag.StringVar(&output, "output", "result.csv", "Output format")
	flag.StringVar(&account, "account", "", "")

	flag.Parse()

	client := New(os.Getenv("MONO_API_KEY"))

	acc, err := client.FindAccount(context.Background(), account)
	if err != nil {
		log.Fatal(err)
	}

	txs, err := client.mono.Transactions(context.Background(), acc.ID, from.date, to.date)
	if err != nil {
		log.Fatal(err)
	}

	ext := filepath.Ext(output)
	if ext != ".csv" {
		for i := 0; i < len(txs); i++ {
			tx := txs[len(txs)-1-i]

			amount := decimal.NewFromInt(tx.Amount)
			amount = amount.Shift(-2)

			fmt.Printf("Number: %d\n", i+1)
			fmt.Printf("ID: %s\n", tx.ID)
			fmt.Printf("Amount: %s UAH\n", amount.String())
			fmt.Printf("Description: %s\n", tx.Description)
			fmt.Printf("Date: %s\n", tx.Time.Format(time.RFC3339))
			fmt.Printf("MCC: %d\n", tx.MCC)
			fmt.Printf("Amount: %0.2f UAH\n", float64(tx.Balance)/100.0)
			fmt.Printf("MCC: %d\n", tx.MCC)
			fmt.Println()
		}

		return
	}

	csvFile, err := os.Create(output)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer csvFile.Close()

	csvwriter := csv.NewWriter(csvFile)

	for _, tx := range txs {
		ccy, err := mono.CurrencyFromISO4217(acc.CurrencyCode)
		if err != nil {
			log.Fatalf("failed to parse currency: %s", err)
		}

		amount := decimal.NewFromInt(tx.Amount)
		amount = amount.Shift(-2)

		row := []string{
			tx.Time.Format("02/01/2006 15:04"),
			tx.Description,
			amount.String(),
			// strconv.FormatInt(int64(tx.MCC), 10),
			ccy.Code,
			mccToCategory(tx),
		}

		if err := csvwriter.Write(row); err != nil {
			log.Fatalf("failed to write csv: %s", err)
		}
	}

	csvwriter.Flush()
}
