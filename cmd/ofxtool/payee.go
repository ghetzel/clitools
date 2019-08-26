package main

import (
	"fmt"
	"sort"

	"github.com/ghetzel/pivot/v3"
	"github.com/ghetzel/pivot/v3/dal"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
)

var Payees pivot.Model

type Payee struct {
	ID               string `pivot:",identity"`
	Name             string
	InstitutionQuery string
	AccountQuery     string
	TransactionQuery string
	Rollups          map[string]float64
	client           *Client
}

func (self *Payee) WithClient(client *Client) *Payee {
	self.client = client
	return self
}

func (self *Payee) Sync(fast bool) error {
	if transactions, err := self.Transactions(); err == nil {
		amounts := make([]float64, len(transactions))
		rollups := make(map[string]float64)

		for i, txn := range transactions {
			// invert the signs because for statistics purposes,
			// we're interested in seeing debits as positive values
			amounts[i] = -1 * txn.Amount
		}

		sort.Float64s(amounts)

		rollups[`Count`] = float64(len(transactions))
		rollups[`Total`] = floats.Sum(amounts)
		rollups[`Minimum`] = floats.Min(amounts)
		rollups[`Maximum`] = floats.Max(amounts)
		rollups[`Median`] = stat.Quantile(0.5, stat.Empirical, amounts, nil)
		rollups[`Mode`], rollups[`ModeCount`] = stat.Mode(amounts, nil)
		rollups[`Mean`], rollups[`StandardDeviation`] = stat.MeanStdDev(amounts, nil)

		self.Rollups = rollups

		return Payees.Update(self)
	} else {
		return err
	}
}

func (self *Payee) Transactions() ([]*Transaction, error) {
	var transactions []*Transaction

	if self.client == nil {
		return nil, fmt.Errorf("payee: Client must be provided")
	}

	if institutions, err := self.client.Institutions(self.InstitutionQuery); err == nil {
		for _, institution := range institutions {
			if accounts, err := institution.Accounts(self.AccountQuery); err == nil {
				for _, account := range accounts {
					if txns, err := account.Transactions(self.TransactionQuery); err == nil {
						transactions = append(transactions, txns...)
					} else {
						return nil, fmt.Errorf("transactions: %v", err)
					}
				}
			} else {
				return nil, fmt.Errorf("accounts: %v", err)
			}
		}
	} else {
		return nil, fmt.Errorf("institutions: %v", err)
	}

	return transactions, nil
}

var PayeesSchema = &dal.Collection{
	Name:                   `payees`,
	IdentityField:          `ID`,
	IdentityFieldType:      dal.StringType,
	IdentityFieldFormatter: dal.GenerateUUID,
	Fields: []dal.Field{
		{
			Name:     `Name`,
			Type:     dal.StringType,
			Required: true,
		}, {
			Name: `InstitutionQuery`,
			Type: dal.StringType,
		}, {
			Name: `AccountQuery`,
			Type: dal.StringType,
		}, {
			Name: `TransactionQuery`,
			Type: dal.StringType,
		}, {
			Name: `Rollups`,
			Type: dal.ObjectType,
		},
	},
}
