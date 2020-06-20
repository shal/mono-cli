package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/shal/mono"
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

func (w *MonoWrap) FindMainUAH(ctx context.Context) (*mono.Account, error) {
	user, err := w.mono.User(ctx)
	if err != nil {
		return nil, err
	}

	for i, acc := range user.Accounts {
		if acc.Type == mono.Black || acc.Type == mono.Platinum {
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

	return nil, errors.New("failed")
}

func main() {
	to := DateFlag{date: time.Now()}
	from := DateFlag{to.date.Add(-24 * 31 * time.Hour)}

	flag.Var(&from, "from", "Start time")
	flag.Var(&to, "to", "Finish time")

	flag.Parse()

	client := New(os.Getenv("MONO_API_KEY"))

	acc, err := client.FindMainUAH(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	txs, err := client.mono.Transactions(context.Background(), acc.ID, from.date, to.date)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(txs); i++ {
		tx := txs[len(txs)-1-i]

		fmt.Printf("Number: %d\n", i+1)
		fmt.Printf("ID: %s\n", tx.ID)
		fmt.Printf("Amount: %0.2f UAH\n", float64(tx.Amount)/100.0)
		fmt.Printf("Description: %s\n", tx.Description)
		fmt.Printf("Date: %s\n", time.Unix(int64(tx.Time), 0).Format(time.RFC822))
		fmt.Printf("MCC: %d\n", tx.MCC)
		fmt.Printf("Amount: %0.2f UAH\n", float64(tx.Balance)/100.0)
		fmt.Println()
	}
}
