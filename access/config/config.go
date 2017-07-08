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
 *     Initial: 2017/07/08        Feng Yife
 */

package config

import (
	"hypercube/libs/log"

	"github.com/spf13/viper"
)

var GNodeConfig *NodeConfig

// NodeConfig is used to config the access endpoint
type NodeConfig struct {
	Addrs             string
	WSReadBufferSize  int
	WSWriteBufferSize int
	PprofAddrs        string
	PrometheusPort    string
	Address           string
	RedisAddress      string
	SessionSecret     string
	SessionName       string
	CorsHosts         []string
	SecretKey         string
	AccessHeartRate   int
	QueueBuffer       int
}

// Load loads the configuration.
func Load() *NodeConfig {
	viper.AddConfigPath("./")
	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		log.Logger.Error("Read configuration file with error:", err)
		panic(err)
	}

	GNodeConfig = &NodeConfig{
		Addrs:             viper.GetString("addrs"),
		WSReadBufferSize:  viper.GetInt("websocket.readBufferSize"),
		WSWriteBufferSize: viper.GetInt("websocket.writeBufferSize"),
		PprofAddrs:        viper.GetString("monitor.pprofAddrs"),
		PrometheusPort:    viper.GetString("monitor.prometheusPort"),
		Address:           viper.GetString("server.address"),
		RedisAddress:      viper.GetString("middleware.session.address"),
		SessionSecret:     viper.GetString("middleware.session.secret"),
		SessionName:       viper.GetString("middleware.session.name"),
		CorsHosts:         viper.GetStringSlice("middleware.cors.hosts"),
		SecretKey:         viper.GetString("middleware.secretkey"),
		AccessHeartRate:   viper.GetInt("accessHeartRate"),
		QueueBuffer:       viper.GetInt("queueBuffer"),
	}

	return GNodeConfig
}
