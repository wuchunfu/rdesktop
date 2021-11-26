package x11

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
)

// Client x11 client
type Client struct {
	conn net.Conn
	info setupInfo
}

type keyCode byte
type window uint32
type colorMap uint32

type format struct {
	depth        byte
	bitsPerPixel byte
	scanlinePad  byte
}

type visual struct {
	id              uint32
	class           byte
	bitsPerRGBValue byte
	colorMapEntries uint16
	redMask         uint32
	greenMask       uint32
	blueMask        uint32
}

type depth struct {
	depth      byte
	numVisuals uint16
	visuals    []visual
}

type screen struct {
	root                window
	defaultColorMap     colorMap
	whitePixel          uint32
	blackPixel          uint32
	currentInputMasks   uint32
	widthInPixels       uint16
	heightInPixels      uint16
	widthInMillimeters  uint16
	heightInMillimeters uint16
	minInstalledMaps    uint16
	maxInstalledMaps    uint16
	rootVisual          uint32
	backingStores       byte
	saveUnders          byte
	rootDepth           byte
	numAllowedDepth     byte
	allowedDepth        []depth
}

type setupInfo struct {
	releaseNumber            uint32
	resourceIdBase           uint32
	resourceIdMask           uint32
	motionBufferSize         uint32
	vendorLen                uint16
	maximumRequestLength     uint16
	screenLen                byte
	pixmapFormatsLen         byte
	imageByteOrder           byte
	bitmapFormatBitOrder     byte
	bitmapFormatScanlineUnit byte
	bitmapFormatScanlinePad  byte
	minKeycode               keyCode
	maxKeycode               keyCode
	vendor                   string
	pixmapFormats            []format
	roots                    []screen
}

// New new client
func New() (*Client, error) {
	disp := os.Getenv("DISPLAY")
	idx := strings.TrimPrefix(disp, ":")
	conn, err := net.Dial("unix", path.Join("/tmp", ".X11-unix", fmt.Sprintf("X%s", idx)))
	if err != nil {
		return nil, err
	}
	cli := &Client{
		conn: conn,
	}
	err = cli.handshake()
	if err != nil {
		conn.Close()
		return nil, err
	}
	return cli, nil
}

// Close close client
func (cli *Client) Close() {
	cli.conn.Close()
}

// https://www.x.org/releases/X11R7.6/doc/xproto/x11protocol.html#connection_setup
func (cli *Client) handshake() error {
	auth, err := readAuth()
	if err != nil {
		return err
	}
	buf := make([]byte, 12+pad(len(auth.name))+pad(len(auth.data)))
	buf[0] = 0x42                           // big endian
	binary.BigEndian.PutUint16(buf[2:], 11) // major
	binary.BigEndian.PutUint16(buf[6:], uint16(len(auth.name)))
	binary.BigEndian.PutUint16(buf[8:], uint16(len(auth.data)))
	copy(buf[12:], auth.name)
	copy(buf[12+pad(len(auth.name)):], auth.data)
	_, err = cli.conn.Write(buf)
	if err != nil {
		return err
	}
	return cli.waitHandshake()
}

func (cli *Client) waitHandshake() error {
	hdr := make([]byte, 8)
	_, err := io.ReadFull(cli.conn, hdr)
	if err != nil {
		return err
	}
	switch hdr[0] {
	case 0: // failed
		buf := make([]byte, hdr[1])
		_, err = io.ReadFull(cli.conn, buf)
		if err != nil {
			return err
		}
		return fmt.Errorf("handshake failed: %s", string(buf))
	case 2: // authenticate
		l := binary.BigEndian.Uint16(hdr[6:])
		buf := make([]byte, l)
		_, err = io.ReadFull(cli.conn, buf)
		if err != nil {
			return err
		}
		return fmt.Errorf("handshake authenticate: %s", string(buf))
	case 1: // success
		l := binary.BigEndian.Uint16(hdr[6:])
		buf := make([]byte, l)
		_, err = io.ReadFull(cli.conn, buf)
		if err != nil {
			return err
		}
		cli.parseSetupInfo(buf)
		return nil
	}
	return nil
}

func pad(n int) int {
	return (n + 3) &^ 3
}

func (cli *Client) parseSetupInfo(data []byte) {
	offset := 0

	cli.info.releaseNumber = binary.BigEndian.Uint32(data[offset:])
	offset += 4

	cli.info.resourceIdBase = binary.BigEndian.Uint32(data[offset:])
	offset += 4

	cli.info.resourceIdMask = binary.BigEndian.Uint32(data[offset:])
	offset += 4

	cli.info.motionBufferSize = binary.BigEndian.Uint32(data[offset:])
	offset += 4

	cli.info.vendorLen = binary.BigEndian.Uint16(data[offset:])
	offset += 2

	cli.info.maximumRequestLength = binary.BigEndian.Uint16(data[offset:])
	offset += 2

	cli.info.screenLen = data[offset]
	offset++

	cli.info.pixmapFormatsLen = data[offset]
	offset++

	cli.info.imageByteOrder = data[offset]
	offset++

	cli.info.bitmapFormatBitOrder = data[offset]
	offset++

	cli.info.bitmapFormatScanlineUnit = data[offset]
	offset++

	cli.info.bitmapFormatScanlinePad = data[offset]
	offset++

	cli.info.minKeycode = keyCode(data[offset])
	offset++

	cli.info.maxKeycode = keyCode(data[offset])
	offset++

	offset += 4 // unused

	vendor := make([]byte, cli.info.vendorLen)
	copy(vendor, data[offset:])
	cli.info.vendor = string(vendor)
	offset += int(cli.info.vendorLen)
	offset = pad(offset)

	cli.parsePixmapFormats(data[offset:], cli.info.pixmapFormatsLen)

	fmt.Println(cli.info)
}

func (cli *Client) parsePixmapFormats(data []byte, n byte) {
	offset := 0

	for i := byte(0); i < n; i++ {
		var fmt format

		fmt.depth = data[offset]
		offset++

		fmt.bitsPerPixel = data[offset]
		offset++

		fmt.scanlinePad = data[offset]
		offset++

		offset += 5 // unused

		cli.info.pixmapFormats = append(cli.info.pixmapFormats, fmt)
	}
}