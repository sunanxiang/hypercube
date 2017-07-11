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
 *     Initial: 2017/07/11        Sun Anxiang
 */

package main

import (
	"hypercube/libs/message"
	"hypercube/orm/cockroach"
	"hypercube/libs/log"
	model "hypercube/model"

	"net/rpc"
	"github.com/jinzhu/gorm"
	"encoding/json"
	"time"
)

type MessageManager int

var (
	Queue       chan message.Message
	Shutdown    chan struct{}
)

func init() {
	Queue = make(chan message.Message, 100)
	Shutdown = make(chan struct{})

	msgManager := new(MessageManager)
	rpc.Register(msgManager)
	rpc.HandleHTTP()

	QueueStart()
}

func (m *MessageManager) Add(msg message.Message, reply *bool) error {
	Queue <- msg
	*reply = true

	return nil
}

func QueueStart() {
	go func() {
		for {
			select {
			case msg := <-Queue:
				HandleMessage(&msg)
			case <-Shutdown:
				return
			}
		}
	}()
}

func HandleMessage(msg *message.Message){
	switch msg.Type {
	case message.MessageTypePlainText:
		HandlePlainText(msg)
	default:
		log.Logger.Debug("Not recognized message type!")
	}
}

func HandlePlainText(msg *message.Message) {
	var content message.PlainText

	json.Unmarshal(msg.Content, &content)

	conn, err := cockroach.DbConnPool.GetConnection()
	if err != nil {
		log.Logger.Error("Get cockroach connect error:", err)
		Queue <- msg

		return
	}
	defer cockroach.DbConnPool.ReleaseConnection(conn)

	db := conn.(*gorm.DB).Exec("SET DATABASE = message")

	dbmsg := model.Message{
		Source:     content.From,
		Target:     content.To,
		Type:       msg.Type,
		IsSend:     false,
		Content:    content.Content,
		Created:    time.Now(),
	}

	err = db.Create(&dbmsg).Error()

	if err != nil {
		log.Logger.Error("Insert into message error:", err)
		Queue <- msg

		return
	}
}