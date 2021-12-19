package internal

import (
	"encoding/json"
	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

type HttpServer struct {
	server        http.Server
	userService   *UserService
	logger        *zap.SugaredLogger
	jwtSigningKey []byte
}

func NewHttpServer(addr string, userService *UserService, jwtSigningKey []byte, logger *zap.SugaredLogger) *HttpServer {
	srv := http.Server{
		Addr: addr,
	}

	httpServer := HttpServer{
		server:        srv,
		userService:   userService,
		logger:        logger,
		jwtSigningKey: jwtSigningKey,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/create_user", httpServer.createUser)
	mux.HandleFunc("/auth_user", httpServer.authUser)
	httpServer.server.Handler = mux

	return &httpServer
}

type createUserRequest struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type createUserResponse struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username,omitempty"`
}

func (c *HttpServer) ListenAndServe() error {
	return c.server.ListenAndServe()
}

func (c *HttpServer) Close() error {
	return c.server.Close()
}

func (c *HttpServer) createUser(rw http.ResponseWriter, r *http.Request) {
	c.logger.Infof("got request for create user")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(500)
		c.logger.Errorf("create user error: error reading user body %v", err)
		return
	}

	var createUserRequest createUserRequest
	err = json.Unmarshal(body, &createUserRequest)
	if err != nil {
		rw.WriteHeader(500)
		c.logger.Errorf("create user error: error unmarshalling request body %v", err)
		return
	}

	user, err := c.userService.createUser(createUserRequest.Username, createUserRequest.Password)
	if err != nil {
		if err == userAlreadyExists {
			c.logger.Errorf("create user error: user with username %v already exists", createUserRequest.Username)
			http.Error(rw, "user already exists", 400)
			return
		}

		rw.WriteHeader(500)
		return
	}

	createUserResponse := createUserResponse{
		UserID:   user.UserID,
		Username: user.UserName,
	}

	responseBytes, err := json.Marshal(&createUserResponse)
	if err != nil {
		rw.WriteHeader(500)
		return
	}
	rw.WriteHeader(200)
	_, err = rw.Write(responseBytes)
	if err != nil {
		c.logger.Errorf("create user error: error writing response %v", err)
	}
}

type authUserRequest struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type authUserResponse struct {
	Token string `json:"token"`
}

func (c *HttpServer) authUser(rw http.ResponseWriter, r *http.Request) {
	c.logger.Infof("got request for auth user")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(500)
		c.logger.Errorf("auth user error: error reading user body %v", err)
		return
	}

	var authUserRequest authUserRequest
	err = json.Unmarshal(body, &authUserRequest)
	if err != nil {
		rw.WriteHeader(500)
		c.logger.Errorf("auth user error: error unmarshalling request body %v", err)
		return
	}

	user, err := c.userService.authUser(authUserRequest.Username, authUserRequest.Password)
	if err != nil {
		c.logger.Errorf("auth user error: %v", err)
		if err == wrongPassword {
			http.Error(rw, "wrong password", 400)
			return
		}
		rw.WriteHeader(500)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.UserName,
		"user_id":  user.UserID,
	})

	tokenString, err := token.SignedString(c.jwtSigningKey)
	if err != nil {
		c.logger.Errorf("auth user error: %v", err)
		rw.WriteHeader(500)
		return
	}

	authUserResponse := authUserResponse{
		Token: tokenString,
	}

	responseBytes, err := json.Marshal(&authUserResponse)
	if err != nil {
		rw.WriteHeader(500)
		return
	}
	rw.WriteHeader(200)
	_, err = rw.Write(responseBytes)
	if err != nil {
		c.logger.Errorf("auth user error: error writing response %v", err)
	}
}
