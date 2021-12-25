package internal

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

type HttpServer struct {
	server        http.Server
	leaseService  *LeaseService
	logger        *zap.SugaredLogger
	jwtSigningKey []byte
}

func NewHttpServer(addr string, leaseService *LeaseService, logger *zap.SugaredLogger, jwtSecret []byte) *HttpServer {
	srv := http.Server{
		Addr: addr,
	}

	httpServer := HttpServer{
		server:        srv,
		leaseService:  leaseService,
		logger:        logger,
		jwtSigningKey: jwtSecret,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/create_lease", httpServer.createLease)
	mux.HandleFunc("/check_lease", httpServer.checkCar)
	httpServer.server.Handler = mux

	return &httpServer
}

type createLeaseRequest struct {
	CarID uint64 `json:"car_id"`
	From  uint64 `json:"from_day"`
	To    uint64 `json:"to_day"`
}

type createLeaseResponse struct {
	UserID  uint64 `json:"user_id"`
	CarID   uint64 `json:"car_id"`
	LeaseID uint64 `json:"lease_id"`
	From    uint64 `json:"from_day"`
	To      uint64 `json:"to_day"`
}

type CheckCarRequest struct {
	CarID uint64 `json:"car_id"`
	From  uint64 `json:"from_day"`
	To    uint64 `json:"to_day"`
}

type CheckCarResponse struct {
	IsFree bool `json:"is_free"`
}

func (c *HttpServer) ListenAndServe() error {
	return c.server.ListenAndServe()
}

type UserAuthObject struct {
	Username string
	UserID   uint64
}

func (c *HttpServer) Close() error {
	return c.server.Close()
}

func (c *HttpServer) createLease(rw http.ResponseWriter, r *http.Request) {
	c.logger.Infof("got request for create lease")
	token := r.Header.Get("X-Auth")
	userAuth, err := c.checkAuth(token)
	if err != nil {
		c.logger.Errorf("auth error: %v", err)
		http.Error(rw, err.Error(), http.StatusUnauthorized)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(500)
		c.logger.Errorf("create lease error: error reading body %v", err)
		return
	}

	var createLeaseRequest createLeaseRequest
	err = json.Unmarshal(body, &createLeaseRequest)
	if err != nil {
		rw.WriteHeader(500)
		c.logger.Errorf("create lease error: error unmarshalling request body %v", err)
		return
	}

	c.logger.Infof("create lease request %+v", createLeaseRequest)

	lease, err := c.leaseService.createLease(userAuth.UserID, createLeaseRequest.CarID, createLeaseRequest.From, createLeaseRequest.To)
	if err != nil {
		if err == leaseAlreadyExists {
			c.logger.Errorf("create lease error: lease with car_id %v already exists", createLeaseRequest.CarID)
			http.Error(rw, "lease already exists", 400)
			return
		}

		rw.WriteHeader(500)
		return
	}

	createLeaseResponse := createLeaseResponse{
		UserID:  lease.UserID,
		CarID:   lease.CarID,
		LeaseID: lease.LeaseID,
		From:    lease.From,
		To:      lease.To,
	}

	responseBytes, err := json.Marshal(&createLeaseResponse)
	if err != nil {
		rw.WriteHeader(500)
		return
	}
	rw.WriteHeader(200)
	_, err = rw.Write(responseBytes)
	if err != nil {
		c.logger.Errorf("create lease error: error writing response %v", err)
	}
}

func (c *HttpServer) checkAuth(token string) (UserAuthObject, error) {
	tokenObj, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return c.jwtSigningKey, nil
	})
	if err != nil {
		return UserAuthObject{}, fmt.Errorf("error casting user id from token: %w", err)
	}

	claims, ok := tokenObj.Claims.(jwt.MapClaims)
	if !ok {
		return UserAuthObject{}, fmt.Errorf("token %s verification error: error casting claims to map claims", token)
	}
	c.logger.Infof("claims %v", claims)
	username := claims["username"].(string)
	userID := claims["user_id"].(float64)

	return UserAuthObject{
		username,
		uint64(userID),
	}, nil
}

func (c *HttpServer) checkCar(rw http.ResponseWriter, r *http.Request) {
	c.logger.Infof("got request for check car")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(500)
		c.logger.Errorf("check lease error: error reading body %v", err)
		return
	}

	var checkCarRequest CheckCarRequest
	err = json.Unmarshal(body, &checkCarRequest)
	if err != nil {
		rw.WriteHeader(400)
		return
	}

	isFree := c.leaseService.IsCarFree(checkCarRequest.CarID, checkCarRequest.From, checkCarRequest.To)

	checkCarResponse := CheckCarResponse{
		IsFree: isFree,
	}

	responseBytes, err := json.Marshal(&checkCarResponse)
	if err != nil {
		rw.WriteHeader(500)
		return
	}
	rw.WriteHeader(200)
	_, err = rw.Write(responseBytes)
	if err != nil {
		c.logger.Errorf("check book error: error writing response %v", err)
	}
}
