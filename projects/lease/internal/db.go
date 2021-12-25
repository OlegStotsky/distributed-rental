package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	badger "github.com/dgraph-io/badger/v3"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

var leaseAlreadyExists = errors.New("lease already exists")
var wrongPassword = errors.New("wrong password")

type LeaseService struct {
	db              *badger.DB
	leaseIDSequence *badger.Sequence
	logger          *zap.SugaredLogger
}

func NewLeaseService(db *badger.DB, leaseIDSequence *badger.Sequence) *LeaseService {
	return &LeaseService{
		db:              db,
		leaseIDSequence: leaseIDSequence,
	}
}

type Lease struct {
	CarID   uint64 `json:"car_id,omitempty"`
	UserID  uint64 `json:"user_id,omitempty"`
	LeaseID uint64 `json:"lease_id,omitempty"`
	From    uint64 `json:"from_day"`
	To      uint64 `json:"to_day"`
}

type LeaseDBModel struct {
	LeaseID uint64 `json:"lease_id,omitempty"`
	CarID   uint64 `json:"car_id,omitempty"`
	UserID  uint64 `json:"user_id,omitempty"`
	From    uint64 `json:"from_day,omitempty"`
	To      uint64 `json:"to_day,omitempty"`
}

func (c *LeaseService) createLease(userID uint64, carID uint64, from, to uint64) (Lease, error) {
	tx := c.db.NewTransaction(true)
	defer tx.Discard()

	leaseID, err := c.leaseIDSequence.Next()
	if err != nil {
		return Lease{}, err
	}

	lease := Lease{
		UserID:  userID,
		CarID:   carID,
		LeaseID: leaseID,
		From:    from,
		To:      to,
	}

	leaseDBModel := LeaseDBModel{
		LeaseID: leaseID,
		UserID:  userID,
		CarID:   carID,
		From:    from,
		To:      to,
	}

	key := fmt.Sprintf("%d_%d_%d", carID, from, to)

	if c.IsCarFree(carID, from, to) {
		leaseBts, err := json.Marshal(&leaseDBModel)
		if err != nil {
			return Lease{}, err
		}
		err = tx.Set([]byte(key), leaseBts)
		if err != nil {
			return Lease{}, err
		}

		err = tx.Commit()
		if err != nil {
			return Lease{}, err
		}

		return lease, nil
	}
	if err == nil {
		return Lease{}, leaseAlreadyExists
	}
	// err != nil and err != ErrKeyNotFound
	return Lease{}, err
}

func (c LeaseService) IsCarFree(carID uint64, from, to uint64) bool {
	carIDStr := strconv.FormatUint(carID, 10)
	tx := c.db.NewTransaction(true)
	defer tx.Discard()
	it := tx.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		if strings.HasPrefix(string(item.Key()), carIDStr) {
			value, err := item.ValueCopy(nil)
			if err != nil {
				return true
			}
			lease := &LeaseDBModel{}
			err = json.Unmarshal(value, lease)
			if err != nil {
				return true
			}
			if lease.From >= from && lease.From <= to {
				return false
			}
			if lease.To >= from && lease.To <= to {
				return false
			}
		}
	}
	return true
}
