// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package binlog

import (
	"fmt"
	"reflect"
	"testing"

	binlogdatapb "github.com/gitql/vitess/go/vt/proto/binlogdata"
	querypb "github.com/gitql/vitess/go/vt/proto/query"
)

var dmlErrorCases = []string{
	"query",
	"query /* _stream 10 (eid id name ) (null 1 'bmFtZQ==' ); */",
	"query /* _stream _table_ eid id name ) (null 1 'bmFtZQ==' ); */",
	"query /* _stream _table_ (10 id name ) (null 1 'bmFtZQ==' ); */",
	"query /* _stream _table_ (eid id name  (null 1 'bmFtZQ==' ); */",
	"query /* _stream _table_ (eid id name)  (null 'aaa' 'bmFtZQ==' ); */",
	"query /* _stream _table_ (eid id name)  (null 'bmFtZQ==' ); */",
	"query /* _stream _table_ (eid id name)  (null 1.1 'bmFtZQ==' ); */",
	"query /* _stream _table_ (eid id name)  (null a 'bmFtZQ==' ); */",
}

func TestEventErrors(t *testing.T) {
	var got *querypb.StreamEvent
	evs := &EventStreamer{
		sendEvent: func(event *querypb.StreamEvent) error {
			got = event
			return nil
		},
	}
	for _, sql := range dmlErrorCases {
		trans := &binlogdatapb.BinlogTransaction{
			Statements: []*binlogdatapb.BinlogTransaction_Statement{
				{
					Category: binlogdatapb.BinlogTransaction_Statement_BL_INSERT,
					Sql:      []byte(sql),
				},
			},
		}
		err := evs.transactionToEvent(trans)
		if err != nil {
			t.Errorf("%s: %v", sql, err)
			continue
		}
		want := &querypb.StreamEvent{
			Statements: []*querypb.StreamEvent_Statement{
				{
					Category: querypb.StreamEvent_Statement_Error,
					Sql:      []byte(sql),
				},
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("error for SQL: '%v' got: %+v, want: %+v", sql, got, want)
		}
	}
}

func TestSetErrors(t *testing.T) {
	evs := &EventStreamer{
		sendEvent: func(event *querypb.StreamEvent) error {
			return nil
		},
	}
	trans := &binlogdatapb.BinlogTransaction{
		Statements: []*binlogdatapb.BinlogTransaction_Statement{
			{
				Category: binlogdatapb.BinlogTransaction_Statement_BL_SET,
				Sql:      []byte("SET INSERT_ID=abcd"),
			},
		},
	}
	before := binlogStreamerErrors.Counts()["EventStreamer"]
	err := evs.transactionToEvent(trans)
	if err != nil {
		t.Error(err)
	}
	got := binlogStreamerErrors.Counts()["EventStreamer"]
	if got != before+1 {
		t.Errorf("got: %v, want: %+v", got, before+1)
	}
}

func TestDMLEvent(t *testing.T) {
	trans := &binlogdatapb.BinlogTransaction{
		Statements: []*binlogdatapb.BinlogTransaction_Statement{{
			Category: binlogdatapb.BinlogTransaction_Statement_BL_SET,
			Sql:      []byte("SET TIMESTAMP=2"),
		}, {
			Category: binlogdatapb.BinlogTransaction_Statement_BL_SET,
			Sql:      []byte("SET INSERT_ID=10"),
		}, {
			Category: binlogdatapb.BinlogTransaction_Statement_BL_INSERT,
			Sql:      []byte("query /* _stream _table_ (eid id name)  (null 1 'bmFtZQ==' ) (null 18446744073709551615 'bmFtZQ==' ); */"),
		}, {
			Category: binlogdatapb.BinlogTransaction_Statement_BL_INSERT,
			Sql:      []byte("query"),
		}},
		EventToken: &querypb.EventToken{
			Timestamp: 1,
			Position:  "MariaDB/0-41983-20",
		},
	}
	evs := &EventStreamer{
		sendEvent: func(event *querypb.StreamEvent) error {
			for _, statement := range event.Statements {
				switch statement.Category {
				case querypb.StreamEvent_Statement_DML:
					want := `category:DML table_name:"_table_" primary_key_fields:<name:"eid" type:INT64 > primary_key_fields:<name:"id" type:UINT64 > primary_key_fields:<name:"name" type:VARBINARY > primary_key_values:<lengths:2 lengths:1 lengths:4 values:"101name" > primary_key_values:<lengths:2 lengths:20 lengths:4 values:"1118446744073709551615name" > `
					got := fmt.Sprintf("%v", statement)
					if got != want {
						t.Errorf("got \n%s, want \n%s", got, want)
					}
				case querypb.StreamEvent_Statement_Error:
					want := `sql:"query" `
					got := fmt.Sprintf("%v", statement)
					if got != want {
						t.Errorf("got %s, want %s", got, want)
					}
				default:
					t.Errorf("unexpected: %#v", event)
				}
			}
			// then test the position
			want := `timestamp:1 position:"MariaDB/0-41983-20" `
			got := fmt.Sprintf("%v", event.EventToken)
			if got != want {
				t.Errorf("got %s, want %s", got, want)
			}
			return nil
		},
	}
	err := evs.transactionToEvent(trans)
	if err != nil {
		t.Error(err)
	}
}

func TestDDLEvent(t *testing.T) {
	trans := &binlogdatapb.BinlogTransaction{
		Statements: []*binlogdatapb.BinlogTransaction_Statement{
			{
				Category: binlogdatapb.BinlogTransaction_Statement_BL_SET,
				Sql:      []byte("SET TIMESTAMP=2"),
			}, {
				Category: binlogdatapb.BinlogTransaction_Statement_BL_DDL,
				Sql:      []byte("DDL"),
			},
		},
		EventToken: &querypb.EventToken{
			Timestamp: 1,
			Position:  "MariaDB/0-41983-20",
		},
	}
	evs := &EventStreamer{
		sendEvent: func(event *querypb.StreamEvent) error {
			for _, statement := range event.Statements {
				switch statement.Category {
				case querypb.StreamEvent_Statement_DDL:
					want := `category:DDL sql:"DDL" `
					got := fmt.Sprintf("%v", statement)
					if got != want {
						t.Errorf("got %s, want %s", got, want)
					}
				default:
					t.Errorf("unexpected: %#v", event)
				}
			}
			// then test the position
			want := `timestamp:1 position:"MariaDB/0-41983-20" `
			got := fmt.Sprintf("%v", event.EventToken)
			if got != want {
				t.Errorf("got %s, want %s", got, want)
			}
			return nil
		},
	}
	err := evs.transactionToEvent(trans)
	if err != nil {
		t.Error(err)
	}
}
