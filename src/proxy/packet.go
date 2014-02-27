package proxy

import (
    "net"
    "io"
    "log"
    "fmt"
)

type PacketIO struct {
    conn net.Conn
    seq uint8
}

var ErrBadConn = fmt.Errorf("connect error")

func (c *PacketIO) ReadPacket() ([]byte, error) {
    // read header, first 3 bytes is length, the fouth is seq num
    header := make([]byte, 4)
    if _, err := io.ReadFull(c.conn, header); err != nil {
        return nil, err
    }

    length := int(uint32(header[0]) | uint32(header[1])<<8 | uint32(header[2])<<16)

    sequence := uint8(header[3])
    log.Println("read length: %d seq %d", length, sequence)

    if sequence != c.seq {
        err := fmt.Errorf("invalid seq number")
        return nil, err
    }

    c.seq++

    data := make([]byte, length)

    if _, err := io.ReadFull(c.conn, data); err != nil {
        return nil, err
    }

    if length < 0x00ffffff {
        return data, nil
    }

    // if length > max payload
    bufdata, err := c.ReadPacket()
    if err != nil {
        return nil, err
    }

    return append(data, bufdata...), nil
}

func (c *PacketIO) WritePacket(data []byte) error {
    buf := make([]byte, len(data) + 4)
    length := len(data)
    copy(buf[4:], data)

    for len(buf) >= 0x00ffffff {
        // TODO not impl packet size > 16M
    }

    buf[0] = byte(length)
    buf[1] = byte(length >> 8)
    buf[2] = byte(length >> 16)
    buf[3] = c.seq

    log.Println("write packet:", buf)
    if n, err := c.conn.Write(buf); err != nil {
        log.Printf("write error %s", err.Error())
        return err
    } else if n != len(buf) {
        return ErrBadConn
    }
    c.seq++
    return nil
}

