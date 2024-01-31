package kafkalistener

import (
	"testing"
)

func TestCompactSchema(t *testing.T) {
	var err error
	var RawSchema, CmpSchema string

	RawSchema = `
		{
			"type": "record",
			"name": "ConnectDefault",
			"namespace": "io.confluent.connect.avro",
			"fields": [
				 {
						"name": "ULTRACLUB_ID",
						"type": "long"
				 },
				 {
						"name": "CURRENT_TIER",
						"type": [
							 "null",
							 "string"
						],
						"default": null
				 },
				 {
						"name": "UPCOMING_TIER",
						"type": [
							 "null",
							 "string"
						],
						"default": null
				 },
				 {
						"name": "UPDATE_DATE",
						"type": [
							 "null",
							 {
									"type": "long",
									"connect.version": 1,
									"connect.name": "org.apache.kafka.connect.data.Timestamp",
									"logicalType": "timestamp-millis"
							 }
						],
						"default": null
				 	}
				]
	 		}
		`

	CmpSchema = `{"type":"record","name":"ConnectDefault","namespace":"io.confluent.connect.avro","fields":[{"name":"ULTRACLUB_ID","type":"long"},{"name":"CURRENT_TIER","type":["null","string"],"default":null},{"name":"UPCOMING_TIER","type":["null","string"],"default":null},{"name":"UPDATE_DATE","type":["null",{"type":"long","connect.version":1,"connect.name":"org.apache.kafka.connect.data.Timestamp","logicalType":"timestamp-millis"}],"default":null}]}`

	schema, err := compactSchema(RawSchema)
	if err != nil {
		t.Errorf("error = %v", err)
	}
	if schema != CmpSchema {
		t.Errorf("Strings not equal = %v", schema)
	}

}
