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

/**
 * Created by HeChengJun on 12/04/2017.
 */

package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"

	"hypercube/proto/general"
)

const (
	userCount = 10
	debugMsg  = false
	Duration = 600
)

var addrs []string = []string{"10.0.0.106:10086", "10.0.0.103:10086"}
var userIDs []uint64 = []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	for i := 0; i < userCount; i++ {
		newRoutine(userIDs[i])
	}
	select {
	case <-interrupt:
		return
	}
}

var i int = 0

func getAddr() string {
    i ++
	return addrs[i % len(addrs)]
}

func newRoutine(from uint64) {
	go testRoutine(getAddr(), from)
}

func randUserID() uint64 {
	return userIDs[rand.Uint32()%userCount]
}

func dial(addr string) (*websocket.Conn, error) {

	u := url.URL{Scheme: "ws", Host: addr, Path: "/join"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

	return c, err
}

func loginPackage(from uint64) []byte {
	message := general.UserAccess{
		UserID: from,
	}
	byteMessage, _ := json.Marshal(message)

	msg := &general.Proto{
		Ver:  general.CurVer,
		Type: general.TypeLoginAccess,
		Body: byteMessage,
	}
	byteMsg, _ := json.Marshal(msg)

	if debugMsg {
		log.Println("login: ", string(byteMsg))
	}

	return byteMsg
}

func testPackage(from, to uint64, t time.Time) []byte {
	message := &general.Message{
		From:    from,
		To:      to,
		Content: t.String(),
	}
	byteMessage, _ := json.Marshal(message)
	msg := &general.Proto{
		Ver:  general.CurVer,
		Type: general.TypeUTUMsg,
		Body: byteMessage,
	}
	byteMsg, _ := json.Marshal(msg)

	if debugMsg {
		log.Println("utu: ", string(byteMsg))
	}

	return byteMsg
}

func writeRoutine(c *websocket.Conn, addr string, from uint64) {
	var msgCount int32 = 0

	// 写入计时
	ticker := time.NewTicker(time.Microsecond * time.Duration(1))
	defer ticker.Stop()

	// 退出计时
	//exitTimer := time.NewTimer(time.Second * time.Duration(rand.Uint32()%60+1))
    exitTimer := time.NewTimer(time.Second * time.Duration(Duration))
	defer exitTimer.Stop()

	for {
		select {
		case t := <-ticker.C:
			// 发送
			to := randUserID()
			message := testPackage(from, to, t)
			err := c.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Println("write:", err)
				goto exit
			}
			msgCount++
		case <-exitTimer.C:
			log.Println("exitTimer : go routine exit, from = ", from)
			goto exit
		}
	}
exit:
	log.Printf("send %d messages, addr %s, from %d \n", msgCount, addr, from)
}

func testRoutine(addr string, from uint64) {
	log.Println("new routine, addr = ",addr ,"userID = ", from)

	// 拨号
	c, err := dial(addr)
	if err != nil {
		log.Println("dial:", err)
		return
	}
	defer c.Close()

	// 发送登录数据包
	message := loginPackage(from)
	err = c.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		log.Println("write:", err)
		return
	}

	// 写
	go writeRoutine(c, addr, from)

	// 读
	exitTimer := time.NewTimer(time.Second * time.Duration(Duration + 10))
	defer exitTimer.Stop()

	var msgCount int32 = 0
	for {
		select {
		case <- exitTimer.C :
			goto exit
		default:

		}
		_, _, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			goto exit
		}
		msgCount++
        if debugMsg {
            log.Printf("to: %d, count: %d, recv: %s \n", from, msgCount, message)
        }
	}
exit:
    log.Printf("recv %d messages, addr = %s, to = %d", msgCount, addr, from)
}
