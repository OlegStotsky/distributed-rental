package internal

import (
	"encoding/json"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

type HttpServer struct {
	server         *http.Server
	bookingService *BookingService
	logger         *zap.SugaredLogger
	jwtSigningKey []byte
}

func NewHttpServer(addr string, bookingService *BookingService, logger *zap.SugaredLogger) *HttpServer {
	srv := &http.Server{
		Addr: addr,
	}

	httpServer := HttpServer{
		server:         srv,
		bookingService: bookingService,
		logger:         logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/create_booking", httpServer.createBooking)
	mux.HandleFunc("/check_car", httpServer.checkCar)

	httpServer.server.Handler = mux

	return &httpServer
}

type createBookingRequest struct {
	UserID uint64 `json:"user_id"`
	CarID  uint64 `json:"car_id"`
	From  uint64 `json:"from_day"`
	To  uint64 `json:"to_day"`
}

type createBookingResponse struct {
	UserID  uint64 `json:"user_id"`
	CarID   uint64 `json:"car_id"`
	BookingID uint64 `json:"booking_id"`
	From  uint64 `json:"from_day"`
	To  uint64 `json:"to_day"`
}

type checkCarRequest struct {
	CarID  uint64 `json:"car_id"`
	From  uint64 `json:"from_day"`
	To  uint64 `json:"to_day"`

}

type checkCarResponse struct {
	IsFree  bool `json:"is_free"`

}

func (c *HttpServer) ListenAndServe() error {
	return c.server.ListenAndServe()
}

func (c *HttpServer) Close() error {
	return c.server.Close()
}

func (c *HttpServer) createBooking(rw http.ResponseWriter, r *http.Request) {
	c.logger.Infof("got request for create booking")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(500)
		c.logger.Errorf("create booking error: error reading body %v", err)
		return
	}

	var createBookingRequest createBookingRequest
	err = json.Unmarshal(body, &createBookingRequest)
	if err != nil {
		rw.WriteHeader(400)
		c.logger.Errorf("create booking error: error unmarshalling request body %v", err)
		return
	}

	booking, err := c.bookingService.createBooking(createBookingRequest.UserID, createBookingRequest.CarID, createBookingRequest.From, createBookingRequest.To)
	if err != nil {
		if err == bookingAlreadyExists {
			c.logger.Errorf("create booking error: booking with car_id %v already exists", createBookingRequest.CarID)
			http.Error(rw, "booking already exists", 400)
			return
		}

		rw.WriteHeader(500)
		return
	}

	createBookingResponse := createBookingResponse{
		UserID:  booking.UserID,
		CarID:   booking.CarID,
		BookingID: booking.BookingID,
		From: booking.From,
		To: booking.To,
	}

	responseBytes, err := json.Marshal(&createBookingResponse)
	if err != nil {
		rw.WriteHeader(500)
		return
	}
	rw.WriteHeader(200)
	_, err = rw.Write(responseBytes)
	if err != nil {
		c.logger.Errorf("create booking error: error writing response %v", err)
	}
}

func (c *HttpServer) checkCar(rw http.ResponseWriter, r *http.Request) {
	c.logger.Infof("got request for check car")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(500)
		c.logger.Errorf("create booking error: error reading body %v", err)
		return
	}

	var checkCarRequest checkCarRequest
	err = json.Unmarshal(body, &checkCarRequest)
	if err != nil {
		rw.WriteHeader(400)
		return
	}

	isFree := c.bookingService.IsCarFree(checkCarRequest.CarID, checkCarRequest.From, checkCarRequest.To)

	checkCarResponse := checkCarResponse{
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
