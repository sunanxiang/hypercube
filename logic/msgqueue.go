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
 *     Modify : 2017/07/28        Yang Chenglong
 */

package main

import (
	"encoding/json"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/fengyfei/hypercube/libs/log"
	"github.com/fengyfei/hypercube/libs/message"
	rp "github.com/fengyfei/hypercube/libs/rpc"
	db "github.com/fengyfei/hypercube/model"
	database "github.com/fengyfei/hypercube/orm"
)

type OfflineMessage chan message.UserEntry

var (
	Queue    chan *message.Message
	Shutdown chan struct{}
	offline  OfflineMessage
)

func initQueue() {
	Queue = make(chan *message.Message, 100)
	Shutdown = make(chan struct{})

	offline = make(chan message.UserEntry)

	QueueStart()
}

func QueueStart() {
	go func() {
		for {
			select {
			case msg := <-Queue:
				HandleMessage(msg)
			case user := <-offline:
				OfflineMessageHandler(user)
			case <-Shutdown:
				close(Queue)
				close(offline)
				return
			}
		}
	}()
}

func OfflineMessageHandler(user message.UserEntry) error {
	mes, err := db.MessageService.GetOffLineMessage(user.UserID.UserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Logger.Debug("User doesn't have offline messages!")
			goto Mess
		}

		log.Logger.Error("GetOffLineMessage Error %v", err)
		return err
	}
Mess:
	for _, msg := range mes {
		switch msg.Type {
		case message.MessageTypePlainText, message.MessageTypeEmotion:
			content := message.PlainText{
				From:    message.User{UserID: msg.Source},
				To:      message.User{UserID: msg.Target},
				Content: msg.Content,
			}

			text, err := json.Marshal(content)
			if err != nil {
				log.Logger.Error("OffLineMessage Marshal Error %v", err)
				return err
			}

			mesg := &message.Message{
				Type:    msg.Type,
				Version: msg.Version,
				Content: text,
			}

			id := msg.Messageid
			flag := TransmitMsg(mesg)
			if flag {
				err := db.MessageService.ModifyMessageStatus(id)
				if err != nil {
					log.Logger.Error("ModifyMessageStatus error:", err)
					ShutDown()
				}
			}
		}
	}

	return nil
}

func HandleMessage(msg *message.Message) {
	switch msg.Type {
	case message.MessageTypePlainText, message.MessageTypeEmotion:
		flag := TransmitMsg(msg)

		HandlePlainText(msg, flag)
	default:
		log.Logger.Debug("Not recognized message type!")
	}
}

func TransmitMsg(msg *message.Message) bool {
	var plainUser message.PlainText

	err := json.Unmarshal(msg.Content, &plainUser)
	if err != nil {
		log.Logger.Error("TransmitMsg Unmarshal Error %v", err)

		return false
	}

	serveIp, flag := onLineUserMag.Query(plainUser.To)
	if flag {
		op := rp.Options{
			Proto: "tcp",
			Addr:  serveIp.ServerIp,
		}

		err := Send(plainUser.To, *msg, op)
		if err != nil {
			log.Logger.Error("TransmitMsg Send Error %v", err)

			return false
		}
	}

	return flag
}

func HandlePlainText(msg *message.Message, isSend bool) {
	var content message.PlainText

	json.Unmarshal(msg.Content, &content)

	dbs := database.Conn

	dbmsg := db.Message{
		Source:  content.From.UserID,
		Target:  content.To.UserID,
		Type:    msg.Type,
		Version: msg.Version,
		IsSend:  isSend,
		Content: content.Content,
		Created: time.Now(),
	}

	err := dbs.Create(&dbmsg).Error

	if err != nil {
		log.Logger.Error("Insert into message error:", err)
		Queue <- msg

		return
	}
}

func ShutDown() {
	Shutdown <- struct{}{}
}
