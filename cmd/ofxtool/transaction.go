package main

import (
	"time"

	"github.com/ghetzel/pivot/v3"
	"github.com/ghetzel/pivot/v3/dal"
)

var Transactions pivot.Model

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
