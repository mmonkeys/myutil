package net

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const MagicNum = 0x1234

type writeData struct {
	writer io.ReadWriter
	data   []byte
}

var writeChan chan *writeData

func init() {
	writeChan = make(chan *writeData, 1024)
	go func() {
		for data := range writeChan {
			magicNum := make([]byte, 2)
			binary.BigEndian.PutUint16(magicNum, MagicNum)
			dataLen := make([]byte, 2)
			binary.BigEndian.PutUint16(dataLen, uint16(len(data.data)))

			buf := bytes.NewBuffer(magicNum)
			buf.Write(dataLen)
			buf.Write(data.data)
			n, err := data.writer.Write(buf.Bytes())
			if err != nil {
				fmt.Printf("write err: %v\n", err)
				continue
			}
			if n != buf.Len() {
				fmt.Printf("write len err: %v, %v\n", n, buf.Len())
				continue
			}
		}
	}()
}

// Write package magic+length+action+content
func Write(writer io.ReadWriter, data []byte) error {
	if writer == nil {
		return nil
	}
	if len(data) > 65535 {
		return errors.New("write too large data")
	}
	writeChan <- &writeData{
		writer: writer,
		data:   data,
	}
	return nil
}

func Read(reader io.ReadWriter) ([]byte, error) {
	peek := make([]byte, 4)

	n, err := io.ReadFull(reader, peek)
	if n != 4 {
		return nil, fmt.Errorf("read peek err %v", peek)
	}
	if err != nil {
		return nil, err
	}
	if binary.BigEndian.Uint16(peek[:2]) != MagicNum {
		return nil, fmt.Errorf("read MagicNum err %v", peek[:2])
	}
	var dataLen uint16
	// 读出 数据包中 实际数据 的长度(大小为 0 ~ 2^16)
	err = binary.Read(bytes.NewReader(peek[2:4]), binary.BigEndian, &dataLen)
	if err != nil {
		return nil, err
	}

	data := make([]byte, dataLen)
	_, err = io.ReadFull(reader, data)
	return data, err
}

func DataPack(action uint8, content []byte) []byte {
	packetBuf := bytes.NewBuffer([]byte{action})
	packetBuf.Write(content)
	return packetBuf.Bytes()
}

func DataPacks(action uint8, content ...[]byte) []byte {
	contentBuf := bytes.NewBuffer(nil)
	for _, c := range content {
		contentLen := make([]byte, 2)
		binary.BigEndian.PutUint16(contentLen, uint16(len(c)))
		contentBuf.Write(contentLen)
		contentBuf.Write(c)
	}
	return DataPack(action, contentBuf.Bytes())
}

func DataUnPack(data []byte) (uint8, []byte) {
	if len(data) < 1 {
		return 0, nil
	}
	return data[0], data[1:]
}

func DataUnPacks(data []byte) [][]byte {
	content := make([][]byte, 0)
	for {
		if len(data) < 2 {
			return nil
		}

		contentLen := binary.BigEndian.Uint16(data[0:2])
		if len(data) < int(contentLen+2) {
			return nil
		}

		content = append(content, data[2:contentLen+2])
		if len(data) == int(contentLen+2) {
			break
		}
		data = data[contentLen+2:]

	}
	return content
}
