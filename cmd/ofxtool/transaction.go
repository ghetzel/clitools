package main

import (
	"math"
	"sort"
	"time"

	"github.com/ghetzel/pivot/v3"
	"github.com/ghetzel/pivot/v3/dal"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
)

var Transactions pivot.Model

type TransactionList []*Transaction

func (self TransactionList) Amounts() (amounts []float64) {
	for _, txn := range self {
		// invert the signs because for statistics purposes,
		// we're interested in seeing debits as positive values
		amounts = append(amounts, -1*txn.Amount)
	}

	sort.Float64s(amounts)
	return
}

func (self TransactionList) Rollup() map[string]interface{} {
	rollups := make(map[string]interface{})
	amounts := self.Amounts()

	rollups[`Count`] = float64(len(self))
	rollups[`Total`] = floats.Sum(amounts)
	rollups[`Minimum`] = floats.Min(amounts)
	rollups[`Maximum`] = floats.Max(amounts)
	rollups[`Median`] = stat.Quantile(0.5, stat.Empirical, amounts, nil)
	rollups[`Mode`], rollups[`ModeCount`] = stat.Mode(amounts, nil)
	mean, stdev := stat.MeanStdDev(amounts, nil)

	if !math.IsNaN(mean) {
		rollups[`Mean`] = mean
	}

	if !math.IsNaN(stdev) {
		rollups[`StandardDeviation`] = stdev
	}

	return rollups
}

type Transaction struct {
	Account        string `pivot:",identity"`
	ID             string
	Description    string
	Memo           string
	Type           string
	Amount         float64
	CorrectionFor  string
	CorrectionType string
	CheckSequence  string
	Reference      string
	SIC            int
	PostedAt       time.Time
	account        *Account
}

var TransactionsSchema = &dal.Collection{
	Name:              `transactions`,
	IdentityField:     `Account`,
	IdentityFieldType: dal.StringType,
	Fields: []dal.Field{
		{
			Name:     `ID`,
			Type:     dal.StringType,
			Key:      true,
			Required: true,
		}, {
			Name:     `Description`,
			Type:     dal.StringType,
			Required: true,
		}, {
			Name:        `Memo`,
			Description: `Additional transaction data.`,
			Type:        dal.StringType,
		}, {
			Name:     `Type`,
			Type:     dal.StringType,
			Required: true,
			Validator: dal.ValidateIsOneOf(
				`credit`,
				`debit`,
				`interest`,
				`dividend`,
				`fee`,
				`service-charge`,
				`deposit`,
				`atm`,
				`point-of-sale`,
				`transfer`,
				`check`,
				`payment`,
				`cash`,
				`direct-deposit`,
				`direct-debit`,
				`repeat-payment`,
				`other`,
			),
		}, {
			Name:         `Amount`,
			Type:         dal.FloatType,
			Required:     true,
			DefaultValue: 0,
		}, {
			Name:         `PostedAt`,
			Type:         dal.TimeType,
			Required:     true,
			DefaultValue: 0,
		}, {
			Name:        `CorrectionFor`,
			Description: `Specifies that this transaction is modifying another.`,
			Type:        dal.StringType,
		}, {
			Name:        `CorrectionType`,
			Description: `Specifies the type of modification this meta-transaction is performing.`,
			Type:        dal.StringType,
			Validator: dal.ValidateIsOneOf(
				nil,
				`delete`,
				`replace`,
			),
		}, {
			Name:        `CheckSequence`,
			Description: `For checks, the check number.`,
			Type:        dal.StringType,
		}, {
			Name:        `Reference`,
			Description: `An institution-specific reference number.`,
			Type:        dal.StringType,
		}, {
			Name:        `SID`,
			Description: `SEC Standard Industry Code`,
			Type:        dal.IntType,
		},
	},
}
