// Copyright 2019-2020 Kosc Telecom.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agent

import (
	"fmt"
	"horus/log"
	"os"
	"strings"
	"time"

	"github.com/vma/glog"
	"github.com/vma/influxclient"
)

// bpoints extends the influxDB BatchPoints
type bpoints struct {
	// reqID is the poll request id
	reqID string

	// res is the poll result
	res PollResult
	influxclient.BatchPoints
}

// InfluxClient is the influxdb (v1) result pusher
type InfluxClient struct {
	// Host is the influx server address
	Host string

	// User is the influx authentication user
	User string

	// Password is the user password
	Password string

	// Database is the influx measurements database
	Database string

	// RetentionPolicy is the retention policy applied to measures
	RetentionPolicy string

	// Timeout is the influx connection and push timeout
	Timeout int

	// WriteRetries is the number of write retries in case of failure
	WriteRetries int

	connected bool
	bpoints   chan bpoints
	influxclient.Client
}

var influxCli *InfluxClient

// NewInfluxClient creates a new influx client and connects to the influx db.
func NewInfluxClient(host, user, passwd, db, rp string, timeout, retries int) error {
	if host == "" || user == "" || passwd == "" || db == "" {
		return fmt.Errorf("influx host, user, password and database must all be defined")
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	if strings.Count(host, ":") <= 1 {
		host += ":8086"
	}
	if rp == "" {
		rp = "autogen"
	}
	influxCli = &InfluxClient{
		Host:            host,
		User:            user,
		Password:        passwd,
		Database:        db,
		RetentionPolicy: rp,
		Timeout:         timeout,
		WriteRetries:    retries,
	}
	return influxCli.dial()
}

// dial connects to the influx server and sends a ping
// to make sure the server is available.
func (c *InfluxClient) dial() error {
	log.Debug2f("connecting to influx %q", c.Host)
	cli, err := influxclient.NewHTTPClient(influxclient.HTTPConfig{
		Addr:      c.Host,
		Username:  c.User,
		Password:  c.Password,
		Timeout:   time.Duration(c.Timeout) * time.Second,
		UserAgent: fmt.Sprintf("snmpagent[%d]", os.Getpid()),
	})
	if err != nil {
		return fmt.Errorf("new http client: %v", err)
	}
	errs := make(chan error)
	go func() {
		_, _, err := cli.Ping(0)
		errs <- err
	}()
	select {
	case <-StopCtx.Done():
		return fmt.Errorf("influx client: dial cancelled")
	case err := <-errs:
		if err != nil {
			return fmt.Errorf("ping: %v", err)
		}
		c.Client = cli
		c.connected = true
		c.bpoints = make(chan bpoints)
		go c.sendData()
		log.Debugf("connected to influx %q", c.Host)
	}
	return nil
}

// Close closes the db connection.
func (c *InfluxClient) Close() {
	c.Client.Close()
	c.connected = false
}

// Push pushes a new poll result to the influx server.
// It actually converts the poll result to influx batchpoints
// and send the latter to the batchpoints channel to be consumed
// by sendData()
func (c *InfluxClient) Push(res PollResult) {
	if c == nil {
		return
	}
	bp, err := c.makeBatchPoints(res)
	if err != nil {
		log.Errorf("influx make batch point: %v, skipping", err)
		return
	}
	log.Debugf("%s - pushing bpoint to influx queue", res.RequestID)
	c.bpoints <- bpoints{res.RequestID, res, bp}
	log.Debug2f("%s - pushed bpoint to influx queue", res.RequestID)
}

// sendData retrieves each new batch point from the influx channel and
// posts it to the influx server. Retries up to WriteRetries times with
// exponential wait time between starting at 1s.
func (c *InfluxClient) sendData() {
	for c.connected {
		select {
		case <-StopCtx.Done():
			glog.Info("cancelled, disconnecting from influx")
			c.Close()
			return
		case bp := <-c.bpoints:
			if len(bp.Points()) == 0 {
				continue
			}
			log.Debug2f("%s - start sending bp to influx", bp.reqID)
			start := time.Now()
			for i := 0; i <= c.WriteRetries; i++ {
				// total write attempts is at worst WriteRetries+1
				if i > 0 {
					time.Sleep(time.Duration(1<<uint(i-1)) * time.Second)
				}
				log.Debug2f("%s - try #%d/%d: writing to influx", bp.reqID, i+1, c.WriteRetries+1)
				if err := c.Write(bp); err != nil {
					log.Errorf("%s - bp len %d, try #%d/%d: influx write: %v", bp.reqID, len(bp.Points()), i+1, c.WriteRetries+1, err)
					if strings.Contains(err.Error(), "partial write") {
						glog.Warningf(">> %s - partial write err, res=%+v", bp.reqID, bp.res)
						break
					}
					continue
				}
				log.Debug2f("%s - try #%d/%d: influx write done in %dms", bp.reqID, i+1, c.WriteRetries+1, time.Since(start)/time.Millisecond)
				break
			}
		}
	}
}

// makeBatchPoints converts a poll result to influx batch points.
func (c *InfluxClient) makeBatchPoints(res PollResult) (influxclient.BatchPoints, error) {
	log.Debug2f("%s - converting to influx batch points", res.RequestID)
	batchPoints, _ := influxclient.NewBatchPoints(influxclient.BatchPointsConfig{
		Database:        c.Database,
		RetentionPolicy: c.RetentionPolicy,
		Precision:       "s",
	})
	for _, scalar := range res.Scalar {
		// each scalar measure is a new influx point
		tags := make(map[string]string)
		for k, v := range res.Tags {
			tags[k] = v
		}
		fields := make(map[string]interface{})
		for _, r := range scalar.Results {
			if !r.ToInflux {
				continue
			}
			fields[r.Name] = r.Value
		}
		pt, err := influxclient.NewPoint(scalar.Name, tags, fields, res.PollStart)
		if err != nil {
			return batchPoints, fmt.Errorf("scalar res %s to point: %v", scalar.Name, err)
		}
		batchPoints.AddPoint(pt)
	}

	for _, indexed := range res.Indexed {
		// each indexed.Results[i] (i.e. all measures for one index) is a new point
		for _, indexedRes := range indexed.Results {
			tags := make(map[string]string)
			for k, v := range res.Tags {
				tags[k] = v
			}
			fields := make(map[string]interface{})
			for _, r := range indexedRes {
				if !r.ToInflux {
					continue
				}
				if r.AsLabel {
					tags[r.Name] = fmt.Sprintf("%v", r.Value)
				} else {
					fields[r.Name] = r.Value
				}
			}
			pt, err := influxclient.NewPoint(indexed.Name, tags, fields, res.PollStart)
			if err != nil {
				return batchPoints, fmt.Errorf("indexed res %s to point: %v", indexed.Name, err)
			}
			batchPoints.AddPoint(pt)
			log.Debug3f(">> indexed pt for `%s`: `%v`", indexed.Name, pt)
		}
	}
	return batchPoints, nil
}
