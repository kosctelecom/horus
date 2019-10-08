package model

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSnmpParams(t *testing.T) {
	tests := []struct {
		in    string
		out   SnmpParams
		valid bool
	}{
		{
			`{
				"ip_address": "10.2.6.79",
				"snmp_version": "2c",
				"snmp_community": "snmpxxx79"
			}`,
			SnmpParams{
				IPAddress:       "10.2.6.79",
				Port:            161,
				Version:         Version2c,
				Community:       "snmpxxx79",
				Timeout:         10,
				Retries:         1,
				ConnectionCount: 1,
			},
			true,
		},
		{
			`{
				"ip_address": "10.2.6.79",
				"snmp_port": 163,
				"snmp_version": "3",
				"snmp_community": "snmpxxx79",
				"snmp_connection_count": 2,
				"snmpv3_security_level": "NoAuthNoPriv"
			}`,
			SnmpParams{
				IPAddress:       "10.2.6.79",
				Port:            163,
				Version:         Version3,
				Community:       "snmpxxx79",
				Timeout:         10,
				Retries:         1,
				ConnectionCount: 2,
				SecLevel:        "NoAuthNoPriv",
			},
			true,
		},
		{
			`{
				"ip_address": "10.2.6.79",
				"snmp_port": 163,
				"snmp_version": "2c"
			}`,
			SnmpParams{},
			false, // no snmp community
		},
	}
	for i, tt := range tests {
		var s SnmpParams
		err := json.Unmarshal([]byte(tt.in), &s)
		valid := err == nil
		if !valid && testing.Verbose() {
			t.Logf("SnmpParams#%d: unmarshal: %v", i, err)
		}
		if valid != tt.valid {
			t.Errorf("SnmpParams#%d: expected validity: %v, got %v", i, tt.valid, valid)
		}
		if !reflect.DeepEqual(s, tt.out) {
			t.Errorf("SnmpParams#%d: expected:\n%+v\ngot:\n%+v\n", i, tt.out, s)
		}
	}
}
