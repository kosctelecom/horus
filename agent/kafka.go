package agent

import (
	"encoding/json"
	"fmt"
	"horus-core/log"
	"os"
	"strings"
	"time"

	"github.com/optiopay/kafka"
	"github.com/optiopay/kafka/proto"
	"github.com/vma/glog"
)

// KafkaClient is a kafka Producer sink for snmp results.
type KafkaClient struct {
	// Host is the kafka server address
	Host string

	// Topic is the kafka topic
	Topic string

	// Partition is the kafka partition number
	Partition int32

	// connected tells wether we are connected to the broker
	connected bool

	// results is the snmp poll results channel
	results chan *PollResult

	broker *kafka.Broker
	kafka.Producer
}

var kafkaCli *KafkaClient

// NewKafkaClient creates a new kafka client and connects to the broker.
func NewKafkaClient(host, topic string, partition int) error {
	if host == "" || topic == "" {
		return fmt.Errorf("kafka host and topic must all be defined")
	}
	if strings.LastIndex(host, ":") == -1 {
		host += ":9092"
	}

	kafkaCli = &KafkaClient{
		Host:      host,
		Topic:     topic,
		Partition: int32(partition),
	}
	return kafkaCli.dial()
}

// dial connects to the kafka broker
func (c *KafkaClient) dial() error {
	log.Debug2f("connecting to kafka %q", c.Host)
	brokerConf := kafka.NewBrokerConf(fmt.Sprintf("snmpagent[%d]", os.Getpid()))
	brokerConf.ReadTimeout = 0 // to avoid unnecessary timeout & reconnections
	errs := make(chan error)
	go func() {
		var err error
		c.broker, err = kafka.Dial([]string{c.Host}, brokerConf)
		errs <- err
	}()
	select {
	case <-StopCtx.Done():
		return fmt.Errorf("kafka client: dial cancelled")
	case err := <-errs:
		if err != nil {
			return fmt.Errorf("kafka dial: %v", err)
		}
		c.connected = true
		producerConf := kafka.NewProducerConf()
		producerConf.RequiredAcks = proto.RequiredAcksLocal
		producerConf.Compression = proto.CompressionGzip
		producerConf.Logger = log.Klogger{}
		c.Producer = c.broker.Producer(producerConf)
		c.results = make(chan *PollResult)
		go c.sendData()
		log.Debugf("connected to kafka %q", c.Host)
	}
	return nil
}

// Close ends the kafka connection
func (c *KafkaClient) Close() {
	c.broker.Close()
	c.connected = false
}

// Push pushes a poll result to the kafka result channel.
func (c *KafkaClient) Push(res *PollResult) {
	if c == nil {
		log.Errorf("kafka client not initialized...")
		return
	}
	log.Debugf("%s: pushing result to kafka queue", res.RequestID)
	c.results <- res
	log.Debug2f("%s: pushed result to kafka queue", res.RequestID)
}

// sendData reads sequentially from kafka channel and writes result to kafka.
func (c *KafkaClient) sendData() {
	for c.connected {
		select {
		case <-StopCtx.Done():
			glog.Info("cancelled, disconnecting from kafka")
			c.Close()
		case res := <-c.results:
			for i := range res.Indexed {
				res.Indexed[i].dedupDesc()
			}
			payload, err := json.Marshal(res)
			if err != nil {
				log.Errorf("%s: poll result marshal: %v", res.RequestID, err)
				continue
			}
			start := time.Now()
			log.Debugf("%s: writing to kafka, payload of %d bytes", res.RequestID, len(payload))
			msg := &proto.Message{Key: []byte(res.RequestID), Value: payload}
			if _, err := c.Produce(c.Topic, c.Partition, msg); err != nil {
				log.Errorf("%s: kafka write: %v", res.RequestID, err)
				continue
			}
			log.Debugf("%s: kafka write done in %dms", res.RequestID, time.Since(start)/time.Millisecond)
		}
	}
}
