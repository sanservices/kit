package kafkalistener

import (
	"errors"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/hamba/avro"
)

var (
	errParseDate        = errors.New("unable to parse date")
	errNoSchemaProvided = errors.New("avro schema not provided")
)

const SimpleDateLayout string = "2006-01-02"

// ParseDate parses a string into a time.Time object.
func ParseDate(datetxt *string) (*time.Time, error) {

	if datetxt == nil {
		return nil, errParseDate
	}

	if len(*datetxt) == 0 {
		return nil, errParseDate
	}

	dt, err := time.Parse(SimpleDateLayout, *datetxt)
	return &dt, err
}

// DecodePayload decodes a message payload into a struct.
func DecodePayload(topic *Topic, payload message.Payload, v interface{}) error {
	if topic.Schema == nil {
		return errNoSchemaProvided
	}

	// try to decode golang-sent messages
	err := avro.Unmarshal(topic.Schema, payload, v)
	if err == nil {
		return nil
	}

	// If the first decode failed try to remove first 5 bytes
	// that corresponds to the kafka schema Id.
	return avro.Unmarshal(topic.Schema, payload[5:], v)
}
