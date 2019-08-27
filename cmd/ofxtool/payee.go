package main

import (
	"fmt"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/pivot/v3"
	"github.com/ghetzel/pivot/v3/dal"
)

var Payees pivot.Model

type Payee struct {
	ID               string `pivot:",identity"`
	Name             string
	InstitutionQuery string
	AccountQuery     string
	TransactionQuery string
	Rollups          map[string]interface{}
	client           *Client
}

func (self *Payee) WithClient(client *Client) *Payee {
	self.client = client
	return self
}

func (self *Payee) Sync(fast bool) error {
	if transactions, err := self.Transactions(); err == nil {
		rollups := transactions.Rollup()
		histoMonths := make(map[string]TransactionList)
		histoYears := make(map[string]TransactionList)

		for _, txn := range transactions {
			histoMonths[txn.PostedAt.Format(`2006-01`)] = append(histoMonths[txn.PostedAt.Format(`2006-01`)], txn)
			histoYears[txn.PostedAt.Format(`2006`)] = append(histoYears[txn.PostedAt.Format(`2006`)], txn)
		}

		byMonth := make([]map[string]interface{}, 0)
		byYear := make([]map[string]interface{}, 0)

		for ym, txns := range histoMonths {
			m, _ := maputil.Merge(txns.Rollup(), map[string]interface{}{
				`ID`: ym,
			})

			byMonth = append(byMonth, m)
		}

		for y, txns := range histoYears {
			m, _ := maputil.Merge(txns.Rollup(), map[string]interface{}{
				`ID`: y,
			})

			byYear = append(byYear, m)
		}

		rollups[`Group`] = map[string]interface{}{
			`ByMonth`: byMonth,
			`ByYear`:  byYear,
		}

		self.Rollups = rollups

		return Payees.Update(self)
	} else {
		return err
	}
}

func (self *Payee) Transactions() (TransactionList, error) {
	var transactions TransactionList

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
