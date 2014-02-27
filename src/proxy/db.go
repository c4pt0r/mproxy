package proxy

import "fmt"

type OkResult struct {
	Status       uint16
	AffectedRows uint64
	LastInsertId uint64
	Info         string
}

type MySQLError struct {
	Errcode uint16
	State   string
	Info    string
}

func (e *MySQLError) Error() string {
	return fmt.Sprintf("MySQL Error %d (%s): %s ", e.Errcode, e.State, e.Info)
}
