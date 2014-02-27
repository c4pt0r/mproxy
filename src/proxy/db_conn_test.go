package proxy

import (
	"testing"
)

func Test_DbConn_Conn(t *testing.T) {
	cfg := NewConfig()
	dbConn := NewDbConn(cfg)
	if dbConn == nil {
		t.Error("mysql conn error")
	} else {
		if err := dbConn.Ping(); err != nil {
			t.Error("ping error")
		}
		if err := dbConn.Prepare("select * from dsb;"); err != nil {
			t.Error("prepare error ", err.Error())
		}
	}
}
