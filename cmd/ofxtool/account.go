package main

import (
	"github.com/ghetzel/pivot/v3"
	"github.com/ghetzel/pivot/v3/dal"
)

var Accounts pivot.Model

type Account struct {
	Institution string `pivot:",identity"`
	AccountID   string
	Name        string
	Type        string
	Balance     float64
	institution *Institution
}

func (self *Account) Resync() error {
	return nil
}

func (self *Account) Transactions() ([]*Transaction, error) {
	var txns []*Transaction

	if err := Transactions.All(&txns); err == nil {
		for _, txn := range txns {
			txn.account = self
		}

		return txns, nil
	} else {
		return nil, err
	}
}

func (self *Account) Transaction(id string) (*Transaction, error) {
	var txn Transaction

	if err := Transactions.Get(id, &txn); err == nil {
		txn.account = self

		return &txn, nil
	} else {
		return nil, err
	}
}

var AccountsSchema = &dal.Collection{
	Name:              `accounts`,
	IdentityField:     `Institution`,
	IdentityFieldType: dal.StringType,
	Fields: []dal.Field{
		{
			Name:     `Account`,
			Type:     dal.StringType,
			Key:      true,
			Required: true,
		}, {
			Name:     `Name`,
			Type:     dal.StringType,
			Required: true,
		}, {
			Name:     `Type`,
			Type:     dal.StringType,
			Required: true,
			Validator: dal.ValidateIsOneOf(
				`checking`,
				`savings`,
				`brokerage`,
				`retirement`,
				`other`,
			),
		}, {
			Name:         `Balance`,
			Type:         dal.FloatType,
			Required:     true,
			DefaultValue: 0,
		},
	},
}
