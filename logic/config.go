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
 *     Initial: 2017/04/04        Feng Yifei
 */

package main

import (
	"hypercube/common/log"
	"github.com/spf13/viper"
)

var (
	logger *log.S8ELogger = log.S8ECreateLogger(
		&log.S8ELogTag{
			log.LogTagService: "Logic Layer",
			log.LogTagType: "common",
		},
		log.S8ELogLevelDefault)

	configuration *LogicLayerConfig
)

// 配置文件结构
type LogicLayerConfig struct {
	NatsUrl             string
	ApiChannel          string
	PprofAddrs          string
	PrometheusPort      string
}

// 初始化配置
func readConfiguration() {
	viper.AddConfigPath("./")
	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		logger.Error("Read configuration file with error:", err)
		panic(err)
	}

	configuration = &LogicLayerConfig{
		NatsUrl:               viper.GetString("nats.urls"),
		ApiChannel:            viper.GetString("nats.apiChannel"),
		PprofAddrs:            viper.GetString("monitoring.pprofAddrs"),
		PrometheusPort:        viper.GetString("monitoring.prometheusPort"),
	}
}

