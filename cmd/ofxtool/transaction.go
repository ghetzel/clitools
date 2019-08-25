package main

import (
	"github.com/ghetzel/pivot/v3"
	"github.com/ghetzel/pivot/v3/dal"
)

var Transactions pivot.Model

type Transaction struct {
	Account       string `pivot:",identity"`
	TransactionID string
	Description   string
	Type          string
	Amount        float64
	account       *Account
}

var TransactionsSchema = &dal.Collection{
	Name:              `transactions`,
	IdentityField:     `Account`,
	IdentityFieldType: dal.StringType,
	Fields: []dal.Field{
		{
			Name:     `TransactionID`,
			Type:     dal.StringType,
			Key:      true,
			Required: true,
		}, {
			Name:     `Description`,
			Type:     dal.StringType,
			Required: true,
		}, {
			Name:     `Type`,
			Type:     dal.StringType,
			Required: true,
			Validator: dal.ValidateIsOneOf(
				`debit`,
				`credit`,
			),
		}, {
			Name:         `Amount`,
			Type:         dal.FloatType,
			Required:     true,
			DefaultValue: 0,
		},
	},
}
