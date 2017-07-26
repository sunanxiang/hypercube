/*
 * MIT License
 *
 * Copyright (c) 2017 SmartestEE Inc.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

/*
 * Revision History:
 *     Initial: 2017/07/06        Feng Yifei
 *      Modify: 2017/07/08        Liu Jiachang		ModifyFunction
 */

package endpoint

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"hypercube/access/conn"
	"hypercube/access/endpoint/handler"
	"hypercube/access/rpc"
	"hypercube/access/session"
	"hypercube/libs/log"
	"hypercube/libs/message"
	"hypercube/libs/metrics/prometheus"
)

// HTTPServer represents the http server accepts the client websocket connections.
type HTTPServer struct {
	node   *Endpoint
	server *echo.Echo
}

// NewHTTPServer create a http server.
func NewHTTPServer(node *Endpoint) *HTTPServer {
	server := &HTTPServer{
		node:   node,
		server: echo.New(),
	}

	server.server.Use(middleware.Logger())
	server.server.Use(middleware.Recover())

	config := middleware.JWTConfig{
		Claims:      jwt.MapClaims{},
		TokenLookup: "query:" + echo.HeaderAuthorization,
		SigningKey:  []byte(node.Conf.SecretKey),
	}
	server.server.Use(middleware.JWTWithConfig(config))

	server.server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: node.Conf.CorsHosts,
		AllowMethods: []string{echo.GET, echo.POST},
	}))

	server.server.GET("/join", server.serve())

	return server
}

func (server *HTTPServer) serve() echo.HandlerFunc {
	var (
		upgrader *websocket.Upgrader
	)

	upgrader = &websocket.Upgrader{
		ReadBufferSize:  server.node.Conf.WSReadBufferSize,
		WriteBufferSize: server.node.Conf.WSWriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(c echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response().Writer, c.Request(), nil)
		if err != nil {
			log.Logger.Error("Upgrade error!", err)
			return err
		}

		claim := c.Get("user")
		if claim == nil {
			log.Logger.Error("Get Claim Error: %v", err)
			return err
		}

		user := message.User{
			UserID: handler.GetUser(claim.(*jwt.Token)),
		}

		_, exist := server.node.clientHub().Get(user.UserID)
		if exist != false {
			log.Logger.Debug("user already login!")
			return nil
		}

		err = server.NewClient(ws, &user, server.node.clientHub(), session.NewSession(ws, &user, server.node, server.node.Conf.QueueBuffer))
		if err != nil {
			log.Logger.Error("HTTPServer NewClient Error: %v", err)
		}

		return nil
	}
}

func (server *HTTPServer) NewClient(ws *websocket.Conn, user *message.User, hub *conn.ClientHub, session *session.Session) error {
	var (
		msg       message.Message
		err       error
		reply     int
		userEntry message.UserEntry
	)

	client := conn.NewClient(user, server.node.clientHub(), session)
	client.StartHandleMessage()

	log.Logger.Info("Endpoint info: ", server.node.Snapshot())

	userEntry = message.UserEntry{
		UserID:   *user,
		ServerIP: message.Access{ServerIp: server.node.Conf.Addrs},
	}

	RpcClient, _ := rpc.RpcClients.Get(server.node.Conf.LogicAddrs)
	if err != nil {
		log.Logger.Error("Get RpcClients Error: %v", err)
		return err

	}
	err = RpcClient.Call("UserHandler.LoginHandler", userEntry, &reply)
	if err != nil {
		log.Logger.Error("UserHandler.LoginHandler Error: %v", err)
		return err
	}

	hub.Add(user, client)
	prometheus.OnlineUserCounter.Add(1)

	defer func() {
		server.node.clientHub().Remove(user, client)
		prometheus.OnlineUserCounter.Add(-1)
		log.Logger.Info("Endpoint info: ", server.node.Snapshot())

		RpcClient, _ := rpc.RpcClients.Get(server.node.Conf.LogicAddrs)
		err = RpcClient.Call("UserHandler.LogoutHandle", userEntry, &reply)
		if err != nil {
			log.Logger.Error("UserHandler.LogoutHandle Error: %v", err)
		}

		client.Close()
	}()

	for {
		if err = ws.ReadJSON(&msg); err != nil {
			log.Logger.Error("ReadMessage Error: %v", err)
			return err
		} else {
			prometheus.ReceiveMessageCounter.Add(1)
			log.Logger.Debug("ReadJSON msg", msg)
			err = client.Handle(&msg)
			if err != nil {
				log.Logger.Error("Handle Message Error: %v", err)
				return err
			}
		}
	}
}
