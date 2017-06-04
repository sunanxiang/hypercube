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
 *     Initial: 2017/03/29        Feng Yifei
 *	   AddFunction: 2017/04/06    Yusan Kurban
 *     AddEcho: 2017/06/04        Yang Chenglong
 */

package main

import (
	"net/http"
	"github.com/gorilla/websocket"
	"hypercube/proto/general"
	"encoding/json"
	"hypercube/proto/api"
	"time"
	"strings"
	"github.com/labstack/echo"
)

func sendAccessInfo()  {
	var (
		r           api.Reply
		serverinfo  api.Access
	)

	addr := strings.Split(configuration.Addrs, ":")[0]

	serverinfo.ServerIp = &addr
	serverinfo.Subject = &configuration.Subject

	info, _ := json.Marshal(serverinfo)

	err := logicRequester.Request(&api.Request{Type: api.ApiTypeAccessInfo, Content: info}, &r, time.Duration(100) * time.Millisecond)

	if err != nil {
		logger.Error("send access info error:", err, " , address:", configuration.Addrs)
	}

	logger.Debug("send access info to logic:", serverinfo, " received reply:", r.Code)
}

func serveWebSocket(c echo.Context) error {
	var upgrader = &websocket.Upgrader{
		ReadBufferSize:     configuration.WSReadBufferSize,
		WriteBufferSize:    configuration.WSWriteBufferSize,
		CheckOrigin:        func(r *http.Request) bool {
			return true
		},
	}

	logger.Debug("New connection")

	if c.Request().Method != "GET" {
		return c.JSON(http.StatusMethodNotAllowed, "Method Not Allowed")
	}

	connection, err := upgrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		logger.Error("websocket upgrade error:", err)
		return err
	}
	defer connection.Close()

	webSocketConnectionHandler(connection)

	return c.JSON(http.StatusInternalServerError,"serveWebSocket connection error")
}

type handlerFunc func(p interface{},req interface{}) interface{}

func webSocketConnectionHandler(conn *websocket.Conn) {
	var (
		err        error
		p          *general.Proto = &general.Proto{}
		ver        *general.Keepalive = &general.Keepalive{}
		mes        *general.Message = &general.Message{}
		user       *general.UserAccess = &general.UserAccess{}
		v          interface{}
		handler    handlerFunc
		id 	   general.UserKey
		ok 	   bool
	)

	for {
		if err = p.ReadWebSocket(conn); err != nil {
			id, ok = OnLineManagement.GetIDByConnection(conn)
			if ok {
				err = OnLineManagement.OnDisconnect(id)
				if err != nil {
					logger.Error("User Logout failed:", err)
				}
			}
			logger.Error("conn read error:", err)
			break
		}

		logger.Debug("Websocket received message type:", p.Type)

		switch p.Type {
		case general.GeneralTypeKeepAlive:
			v = ver
			handler = keepAliveRequestHandler
		case general.GeneralTypeTextMsg:
			v = mes
			handler = userMessageHandler
		case general.GeneralTypeLogin:
			v = user
		case general.GeneralTypeLogout:
			v = user
		}

		if v != nil {
			err = json.Unmarshal(p.Body, v)

			if err != nil {
				logger.Error("Receive unknown message:", err)
				continue
			} else {
				switch p.Type {
				case general.GeneralTypeLogin:
					user = v.(*general.UserAccess)
					err = OnLineManagement.OnConnect(user.UserID, conn)
					if err != nil {
						logger.Error("User Login failed:", err)
					}
				case general.GeneralTypeLogout:
					user = v.(*general.UserAccess)
					err = OnLineManagement.OnDisconnect(user.UserID)
					if err != nil {
						logger.Error("User Logout failed:", err)
					}
				default:
					handler(p,v)
				}
			}
		}
	}
}
