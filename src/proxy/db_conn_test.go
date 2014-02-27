package proxy

import (
    "testing"
)

func Test_DbConn_Conn(t *testing.T) {
    cfg := NewConfig()
    dbConn := NewDbConn(cfg)
    if dbConn == nil {
        t.Error("mysql conn error")
    }
}
