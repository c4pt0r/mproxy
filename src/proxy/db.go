package proxy

type Result struct {
	status       uint16
	affectedRows uint64
	lastInsertId uint64
	info         string
	errcode      uint16
}
