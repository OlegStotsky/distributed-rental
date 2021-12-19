package internal

import (
	"encoding/json"
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

func NewHttpServer(addr string, leaseService *LeaseService, logger *zap.SugaredLogger) *HttpServer {
	srv := http.Server{
		Addr: addr,
	}

	httpServer := HttpServer{
		server:       srv,
		leaseService: leaseService,
		logger:       logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/create_lease", httpServer.createLease)
	httpServer.server.Handler = mux

	return &httpServer
}

type createLeaseRequest struct {
	UserID uint64 `json:"user_id"`
	CarID  uint64 `json:"car_id"`
}

type createLeaseResponse struct {
	UserID  uint64 `json:"user_id"`
	CarID   uint64 `json:"car_id"`
	LeaseID uint64 `json:"lease_id"`
}

func (c *HttpServer) ListenAndServe() error {
	return c.server.ListenAndServe()
}

func (c *HttpServer) Close() error {
	return c.server.Close()
}

func (c *HttpServer) createLease(rw http.ResponseWriter, r *http.Request) {
	c.logger.Infof("got request for create lease")

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

	lease, err := c.leaseService.createLease(createLeaseRequest.UserID, createLeaseRequest.CarID)
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
