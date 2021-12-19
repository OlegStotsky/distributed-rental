package internal

import (
	"encoding/json"
	"errors"
	badger "github.com/dgraph-io/badger/v3"
	"go.uber.org/zap"
	"strconv"
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
}

type LeaseDBModel struct {
	LeaseID uint64 `json:"lease_id,omitempty"`
	CarID   uint64 `json:"car_id,omitempty"`
	UserID  uint64 `json:"user_id,omitempty"`
}

func (c *LeaseService) createLease(userID uint64, carID uint64) (Lease, error) {
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
	}

	leaseDBModel := LeaseDBModel{
		LeaseID: leaseID,
		UserID:  userID,
		CarID:   carID,
	}

	carIDBts := []byte(strconv.FormatUint(carID, 10))
	_, err = tx.Get(carIDBts)

	if err == badger.ErrKeyNotFound {
		userBts, err := json.Marshal(&leaseDBModel)
		if err != nil {
			return Lease{}, err
		}
		err = tx.Set(carIDBts, userBts)
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
