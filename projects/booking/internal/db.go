package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"go.uber.org/zap"
)

type BookingService struct {
	DB                *badger.DB
	BookingIDSequence *badger.Sequence
	Logger            *zap.Logger
}

type Booking struct {
	CarID     uint64 `json:"car_id,omitempty"`
	UserID    uint64 `json:"user_id,omitempty"`
	BookingID uint64 `json:"booking_id,omitempty"`
	From      uint64 `json:"from_day,omitempty"`
	To        uint64 `json:"to_day,omitempty"`
}

type BookingDBModel struct {
	CarID     uint64 `json:"car_id,omitempty"`
	UserID    uint64 `json:"user_id,omitempty"`
	BookingID uint64 `json:"booking_id,omitempty"`
	From      uint64 `json:"from_day,omitempty"`
	To        uint64 `json:"to_day,omitempty"`
}

var bookingAlreadyExists = errors.New("booking already exists")

func getKey(carID, from, to uint64) []byte {
	return []byte(fmt.Sprintf("%d_%d_%d", carID, from, to))
}

func (c *BookingService) createBooking(userID uint64, carID uint64, from, to uint64) (Booking, error) {
	tx := c.DB.NewTransaction(true)
	defer tx.Discard()

	bookingID, err := c.BookingIDSequence.Next()
	if err != nil {
		return Booking{}, err
	}

	booking := Booking{
		UserID:    userID,
		CarID:     carID,
		BookingID: bookingID,
		From:      from,
		To:        to,
	}

	bookingDBModel := BookingDBModel{
		BookingID: bookingID,
		UserID:    userID,
		CarID:     carID,
		From:      from,
		To:        to,
	}

	carIDBts := getKey(carID, from, to)
	if c.IsCarFree(carID, from, to) {
		userBts, err := json.Marshal(&bookingDBModel)
		if err != nil {
			return Booking{}, err
		}
		err = tx.Set(carIDBts, userBts)
		if err != nil {
			return Booking{}, err
		}

		err = tx.Commit()
		if err != nil {
			return Booking{}, err
		}

		return booking, err
	}
	return Booking{}, bookingAlreadyExists
}

func (c *BookingService) IsCarFree(carID uint64, from, to uint64) bool {
	tx := c.DB.NewTransaction(true)
	defer tx.Discard()
	it := tx.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		value, err := item.ValueCopy(nil)
		if err != nil {
			return true
		}
		booking := &BookingDBModel{}
		err = json.Unmarshal(value, booking)
		if err != nil {
			return true
		}
		if booking.CarID != carID {
			continue
		}
		if booking.From >= from && booking.From <= to {
			return false
		}
		if booking.To >= from && booking.To <= to {
			return false
		}
	}
	return true
}
