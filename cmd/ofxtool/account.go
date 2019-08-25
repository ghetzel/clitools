package main

import (
	"fmt"
	"strings"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/ofxgo"
	"github.com/ghetzel/pivot/v3"
	"github.com/ghetzel/pivot/v3/dal"
)

var Accounts pivot.Model

type Account struct {
	Institution      string `pivot:",identity"`
	ID               string
	Name             string
	Type             string
	Balance          float64
	AvailableBalance float64
	Broker           string
	Routing          string
	institution      *Institution
}

func (self *Account) Sync() error {
	if err := self.syncTransactions(); err != nil {
		return err
	}

	return nil
}

func (self *Account) syncTransactions() error {
	if self.institution == nil {
		return fmt.Errorf("cannot sync account without a parent institution")
	}

	var reqmsg ofxgo.Message

	switch self.Type {
	case `brokerage`:
	case `checking`:
		if self.Routing == `` {
			return fmt.Errorf("[%v] sync: cannot sync checking account without routing number", self.ID)
		}

		reqmsg = &ofxgo.StatementRequest{
			TrnUID:  self.institution.txnID(),
			Include: ofxgo.Boolean(true),
			BankAcctFrom: ofxgo.BankAcct{
				BankID:   ofxgo.String(self.Routing),
				AcctID:   ofxgo.String(self.ID),
				AcctType: ofxgo.AcctTypeChecking,
			},
		}

	default:
		return fmt.Errorf("[%v] sync: unsupported account type %q", self.ID, self.Type)
	}

	if req, err := self.institution.ofxreq(reqmsg); err == nil {
		if res, err := self.institution.ofxdo(req); err == nil {
			var merr error

			for _, msg := range res.Bank {
				if stmt, ok := msg.(*ofxgo.StatementResponse); ok {
					self.Balance, _ = stmt.BalAmt.Float64()
					self.AvailableBalance, _ = stmt.AvailBalAmt.Float64()

					if tlist := stmt.BankTranList; tlist != nil {
						for _, txn := range tlist.Transactions {
							merr = log.AppendError(merr, self.saveTransaction(&txn))
						}
					}
				}
			}

			return merr
		} else {
			return fmt.Errorf("[%v] response: %v", self.ID, err)
		}
	} else {
		return fmt.Errorf("[%v] request: %v", self.ID, err)
	}
}

func (self *Account) saveTransaction(txn *ofxgo.Transaction) error {
	amt, _ := txn.TrnAmt.Float64()
	desc := ``

	if v := txn.Name.String(); v != `` {
		desc = v
	} else if payee := txn.Payee; payee != nil {
		desc = payee.Name.String()
	} else {
		desc = txn.PayeeID.String()
	}

	transaction := Transaction{
		Account:       self.ID,
		ID:            txn.FiTID.String(),
		Description:   desc,
		Memo:          txn.Memo.String(),
		Type:          self.trnTypeToString(txn.TrnType.String()),
		Amount:        amt,
		CheckSequence: txn.CheckNum.String(),
		Reference:     txn.RefNum.String(),
		SIC:           int(txn.SIC),
		PostedAt:      txn.DtPosted.Time,
	}

	if cf := txn.CorrectFiTID.String(); cf != `` {
		transaction.CorrectionFor = cf
		transaction.CorrectionType = strings.ToLower(txn.CorrectAction.String())
	}

	if Transactions.Exists([]string{
		transaction.Account,
		transaction.ID,
	}) {
		return Transactions.Update(&transaction)
	} else {
		return Transactions.Create(&transaction)
	}
}

func (self *Account) trnTypeToString(ofxTT string) string {
	switch ofxTT {
	case `INT`:
		return `interest`
	case `DIV`:
		return `dividend`
	case `SRVCHG`:
		return `service-charge`
	case `DEP`:
		return `deposit`
	case `POS`:
		return `point-of-sale`
	case `XFER`:
		return `transfer`
	case `DIRECTDEP`:
		return `direct-deposit`
	case `DIRECTDEBIT`:
		return `direct-debit`
	default:
		return strings.ToLower(ofxTT)
	}
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
			Name:     `ID`,
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
		}, {
			Name:         `AvailableBalance`,
			Type:         dal.FloatType,
			Required:     true,
			DefaultValue: 0,
		}, {
			Name: `Broker`,
			Type: dal.StringType,
		}, {
			Name: `Routing`,
			Type: dal.StringType,
		},
	},
}
