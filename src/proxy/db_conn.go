package proxy

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"
)

type DbConn struct {
	PacketIO
	closed     bool
	busy       bool
	dbAddr     string
	dbName     string
	dbUser     string
	dbPassword string

	cipher     []byte
	capability uint32
	status     uint16
	lastPing   int64
}

func NewDbConn(cfg *config) *DbConn {
	ret := new(DbConn)
	ret.dbAddr = cfg.dbAddr
	ret.dbName = cfg.dbName
	ret.dbUser = cfg.dbUser
	ret.dbPassword = cfg.dbPassword
	ret.busy = false
	ret.closed = false

	if err := ret.Reconnect(); err != nil {
		return nil
	}
	return ret
}

func (c *DbConn) Reconnect() error {
	c.closed = false
	if c.conn != nil {
		c.conn.Close()
	}

	netConn, err := net.Dial("tcp", c.dbAddr)
	if err != nil {
		log.Printf("connect %s error, %s", c.dbAddr, err.Error())
		return err
	}

	c.conn = netConn
	c.seq = 0

	if err := c.ReadHandshake(); err != nil {
		log.Printf("read server handshake error %s", err.Error())
		return err
	}

	if err := c.WriteAuthPacket(); err != nil {
		log.Printf("write auth packet error %s", err.Error())
		return err
	}
	if _, err := c.ReadOK(); err != nil {
		return err
	}
	return nil
}

func (c *DbConn) ReadHandshake() error {
	data, err := c.ReadPacket()

	if err != nil {
		log.Println("read handshake error %s", err.Error())
		return err
	}

	if data[0] == iERR {
		err := fmt.Errorf("read init handshake error")
		return err
	}

	if data[0] < minProtocolVersion {
		err := fmt.Errorf("too old protocol")
		return err
	}

	// skip mysql version
	pos := 1
	log.Printf("mysql version: [%s]", data[pos:bytes.IndexByte(data[1:], 0x00)])
	pos += bytes.IndexByte(data[1:], 0x00) + 1 + 4

	c.cipher = append(c.cipher, data[pos:pos+8]...)
	pos += 8
	pos += 1

	c.capability = uint32(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2

	if len(data) > pos {
		pos += 1

		c.status = binary.LittleEndian.Uint16(data[pos : pos+2])
		pos += 2

		c.capability = uint32(binary.LittleEndian.Uint16(data[pos:pos+2]))<<16 | c.capability

		pos += 2

		//skip auth data len or [00]
		//skip reserved (all [00])
		pos += 10 + 1

		// steal from go-mysql-driver
		// second part of the password cipher [mininum 13 bytes],
		// where len=MAX(13, length of auth-plugin-data - 8)
		//
		// The web documentation is ambiguous about the length. However,
		// according to mysql-5.7/sql/auth/sql_authentication.cc line 538,
		// the 13th byte is "\0 byte, terminating the second part of
		// a scramble". So the second part of the password cipher is
		// a NULL terminated string that's at least 13 bytes with the
		// last byte being NULL.
		//
		// The official Python library uses the fixed length 12
		// which seems top work but technically could have a hidden bug.
		c.cipher = append(c.cipher, data[pos:pos+12]...)
	}
	return nil
}

func (c *DbConn) WriteAuthPacket() error {
	var clientFlag uint32
	clientFlag = clientProtocol41 | clientSecureConn | clientLongPassword | clientTransactions | clientLongFlag | clientConnectWithDB
	clientFlag &= c.capability

	length := 4 + 4 + 1 + 23
	length += len([]byte(c.dbUser)) + 1

	scramble := c.genPassword([]byte(c.dbPassword))
	length += len(scramble) + 1
	length += len(c.dbName) + 1

	c.capability = clientFlag

	data := make([]byte, length)

	data[0] = byte(clientFlag)
	data[1] = byte(clientFlag >> 8)
	data[2] = byte(clientFlag >> 16)
	data[3] = byte(clientFlag >> 24)

	// leave max packet size 0

	// set charset
	data[8] = byte(collation_utf8_general_ci)
	pos := 9 + 23
	// set username
	pos += copy(data[pos:], []byte(c.dbUser))
	pos += 1

	data[pos] = byte(len(scramble))
	pos += copy(data[pos+1:], scramble)
	pos += 1

	pos += copy(data[pos:], c.dbName)

	return c.WritePacket(data)
}

func (c *DbConn) ReadOK() (*OkResult, error) {
	data, err := c.ReadPacket()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	if data[0] == byte(iOK) {
		return c.ParseOKPacket(data), nil
	} else if data[0] == byte(iERR) {
		return nil, c.ParseErrPacket(data)
	} else {
		err := fmt.Errorf("invalid packet")
		log.Println(err.Error())
		return nil, err
	}
}

func (c *DbConn) ParseOKPacket(b []byte) *OkResult {
	r := new(OkResult)
	pos := 1
	var n int
	r.AffectedRows, _, n = readLengthEncodedInteger(b[pos:])
	pos += n
	r.LastInsertId, _, n = readLengthEncodedInteger(b[pos:])
	pos += n

	r.Status = binary.LittleEndian.Uint16(b[pos:])
	c.status = r.Status

	return r
}

func (c *DbConn) ParseErrPacket(b []byte) error {
	r := new(MySQLError)

	pos := 1

	r.Errcode = binary.LittleEndian.Uint16(b[pos:])
	pos += 2

	pos++
	r.State = string(b[pos : pos+5])
	pos += 5

	r.Info = string(b[pos:])

	return r
}

// write simple command
func (c *DbConn) WriteCmd(cmd byte, args []byte) error {
	c.seq = 0
	if args == nil {
		return c.WritePacket([]byte{
			cmd,
		})
	}
	// if we got arguments
	length := len(args) + 1
	data := make([]byte, length)
	data[0] = cmd
	copy(data[1:], args)
	return c.WritePacket(data)
}

func (c *DbConn) Prepare(query string) error {
	if err := c.WriteCmd(comStmtPrepare, []byte(query)); err != nil {
		return err
	}
	data, err := c.ReadPacket()
	if err != nil {
		return err
	}

	if data[0] != iOK {
		err := c.ParseErrPacket(data)
		log.Println(err.Error())
		return err
	} else {
		result := c.ParseOKPacket(data)
		log.Println("Prepare OK", result.Status)
	}

	log.Println("prepare response packet: ", data)
	return nil
}

func (c *DbConn) Ping() error {
	n := time.Now().Unix()
	if n-c.lastPing > 30 {
		if err := c.WriteCmd(comPing, nil); err != nil {
			return err
		}
		if _, err := c.ReadOK(); err != nil {
			return err
		}
		c.lastPing = n
	}
	return nil
}

func (c *DbConn) genPassword(password []byte) []byte {
	if c.cipher == nil || password == nil {
		return nil
	}

	crypt := sha1.New()
	crypt.Write(password)

	t := crypt.Sum(nil)

	crypt.Reset()
	crypt.Write(t)
	hash := crypt.Sum(nil)

	crypt.Reset()
	crypt.Write(c.cipher)
	crypt.Write(hash)
	scramble := crypt.Sum(nil)

	for i := range scramble {
		scramble[i] ^= t[i]
	}

	return scramble
}
