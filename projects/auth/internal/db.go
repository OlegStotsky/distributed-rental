package internal

import (
	"encoding/json"
	"errors"
	badger "github.com/dgraph-io/badger/v3"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var userAlreadyExists = errors.New("user already exists")
var wrongPassword = errors.New("wrong password")

type UserService struct {
	db             *badger.DB
	userIDSequence *badger.Sequence
	logger         *zap.SugaredLogger
}

func NewUserService(db *badger.DB, userIDSequence *badger.Sequence) *UserService {
	return &UserService{
		db:             db,
		userIDSequence: userIDSequence,
	}
}

type User struct {
	UserID   uint64 `json:"user_id,omitempty"`
	UserName string `json:"user_name,omitempty"`
}

type UserDBModel struct {
	UserID       uint64 `json:"user_id,omitempty"`
	UserName     string `json:"user_name,omitempty"`
	PasswordHash string `json:"password_hash,omitempty"`
}

func (c *UserService) createUser(username string, password string) (User, error) {
	tx := c.db.NewTransaction(true)
	defer tx.Discard()

	userID, err := c.userIDSequence.Next()
	if err != nil {
		return User{}, err
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}

	user := User{
		UserID:   userID,
		UserName: username,
	}

	userDBModel := UserDBModel{
		UserID:       userID,
		UserName:     username,
		PasswordHash: passwordHash,
	}

	userNameBts := []byte(username)
	_, err = tx.Get(userNameBts)
	if err == badger.ErrKeyNotFound {
		userBts, err := json.Marshal(&userDBModel)
		if err != nil {
			return User{}, err
		}
		err = tx.Set(userNameBts, userBts)
		if err != nil {
			return User{}, err
		}

		err = tx.Commit()
		if err != nil {
			return User{}, err
		}

		return user, nil
	}
	if err == nil {
		return User{}, userAlreadyExists
	}
	// err != nil and err != ErrKeyNotFound
	return User{}, err
}

func (c *UserService) authUser(username string, password string) (User, error) {
	tx := c.db.NewTransaction(false)
	defer tx.Discard()

	userNameBts := []byte(username)
	userDBObject, err := tx.Get(userNameBts)

	val, err := userDBObject.ValueCopy(nil)
	if err != nil {
		return User{}, err
	}

	userDBModel := UserDBModel{}
	err = json.Unmarshal(val, &userDBModel)
	if err != nil {
		return User{}, err
	}

	ok := CheckPasswordHash(password, userDBModel.PasswordHash)
	if !ok {
		return User{}, wrongPassword
	}

	return User{
		UserID:   userDBModel.UserID,
		UserName: userDBModel.UserName,
	}, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
