package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/urfave/cli"
	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter
var serviceUUID = baseUUID(0x1234)
var writeUUID = baseUUID(0x6e40)
var notifyUUID = baseUUID(0x6e41)
var advName = "pico2w_ble"

func baseUUID(short uint16) bluetooth.UUID {
	var b = [16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x80, 0x00, 0x00, 0x80, 0x5F, 0x9B, 0x34, 0xFB}
	b[2] = byte(short >> 8)
	b[3] = byte(short & 0xFF)
	return bluetooth.NewUUID(b)
}

type methodType byte

const (
	echo methodType = iota
	uploadImage
	deleteImage
	listImages
	getImage
)

func (m methodType) String() string {
	switch m {
	case echo:
		return "echo"
	case uploadImage:
		return "upload"
	case deleteImage:
		return "delete"
	case listImages:
		return "list"
	case getImage:
		return "get"
	default:
		return "unknown"
	}
}

func main() {
	app := &cli.App{
		Name:  "bleclient",
		Usage: "BLE Client for file operations",
		Commands: []cli.Command{
			{
				Name:   "scan",
				Usage:  "Scan all qualified devices",
				Action: scan,
			},
			{
				Name:   "echo",
				Usage:  "Send echo message",
				Action: runEcho,
			},
			{
				Name:   "upload",
				Usage:  "Upload a file",
				Action: runUpload,
			},
			{
				Name:   "delete",
				Usage:  "Delete a file by MD5",
				Action: runDelete,
			},
			{
				Name:   "list",
				Usage:  "List all files",
				Action: runList,
			},
			{
				Name:   "get",
				Usage:  "Get one file",
				Action: runGetFile,
			},
			{
				Name: "convert",
				Subcommands: cli.Commands{
					{
						Name:   "img",
						Usage:  "Convert one image to album suitable format and raw data",
						Action: convertImage,
					},
					{
						Name:   "raw",
						Usage:  "Convert one raw data to bmp",
						Action: convertRaw,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
}
func scan(c *cli.Context) error {
	err := adapter.Enable()
	if err != nil {
		return err
	}

	return adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if result.LocalName() != advName {
			return
		}
		fmt.Printf("found device: %s, RSSI: %d, %s\n", result.Address.String(), result.RSSI, result.LocalName())
	})
}

func connectToPeripheral() (*bluetooth.Device, *bluetooth.DeviceCharacteristic,
	*bluetooth.DeviceCharacteristic, error) {
	err := adapter.Enable()
	if err != nil {
		return nil, nil, nil, err
	}

	var found bool
	var deviceAddress bluetooth.Address
	err = adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if result.LocalName() != advName {
			return
		}
		fmt.Printf("found device: %s, RSSI: %d, %s\n", result.Address.String(), result.RSSI, result.LocalName())
		deviceAddress = result.Address
		found = true
		adapter.StopScan()
	})
	if err != nil {
		return nil, nil, nil, err
	}

	if !found {
		return nil, nil, nil, errors.New("PicoServer not found")
	}

	device, err := adapter.Connect(deviceAddress, bluetooth.ConnectionParams{})
	if err != nil {
		return nil, nil, nil, err
	}
	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		return nil, nil, nil, err
	}

	writeChars, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{writeUUID})
	if err != nil {
		return nil, nil, nil, err
	}

	notiChars, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{notifyUUID})
	if err != nil {
		return nil, nil, nil, err
	}

	return &device, &writeChars[0], &notiChars[0], nil
}

func writeWithDelay(char *bluetooth.DeviceCharacteristic, data []byte) error {
	_, err := char.WriteWithoutResponse(data)
	time.Sleep(3 * time.Second)
	return err
}

func writeHeader(char *bluetooth.DeviceCharacteristic, m methodType) error {
	return writeWithDelay(char, []byte{byte(m)})
}

func sendRequest(char *bluetooth.DeviceCharacteristic, m methodType, body []byte) error {
	fmt.Printf("Sending method %s and content (%dB)\n", m, len(body))

	return writeWithDelay(char, append([]byte{byte(m)}, body...))
}

func runEcho(c *cli.Context) error {
	device, writeChar, notifyChar, err := connectToPeripheral()
	if err != nil {
		return err
	}
	defer device.Disconnect()

	// Enable notifications before sending the request
	notifyChar.EnableNotifications(func(buf []byte) {
		fmt.Println("Received response:", string(buf))
	})

	time.Sleep(time.Second)

	err = sendRequest(writeChar, echo, []byte("ping!"))
	if err != nil {
		return err
	}

	return sendRequest(writeChar, echo, []byte("pong!"))
}

func prepareFileHeader(md5hex string, v uint32) ([]byte, error) {
	buf := make([]byte, 16)
	_, err := fmt.Sscanf(md5hex, "%x", &buf)
	if err != nil {
		return nil, err
	}

	fmt.Printf("md5sum: %s\n", md5hex)

	buf = append(buf, byte(v>>24))
	buf = append(buf, byte(v>>16))
	buf = append(buf, byte(v>>8))
	buf = append(buf, byte(v))

	return buf, nil
}

func runUpload(c *cli.Context) error {
	filename := c.Args()[0]
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	buf, err := prepareFileHeader(fmt.Sprintf("%x", md5.Sum(content)), uint32(len(content)))
	if err != nil {
		return err
	}

	device, writeChar, notifyChar, err := connectToPeripheral()
	if err != nil {
		return err
	}
	defer device.Disconnect()

	notifyChar.EnableNotifications(func(buf []byte) {
		fmt.Println("Received upload response:", string(buf))
	})

	return sendRequest(writeChar, uploadImage, append(buf, content...))
}

func runDelete(c *cli.Context) error {
	md5hex := c.Args()[0]
	if len(md5hex) != 32 {
		return fmt.Errorf("MD5 must be 32 hex characters")
	}

	size, err := strconv.Atoi(c.Args()[1])
	if err != nil {
		return err
	}

	payload, err := prepareFileHeader(md5hex, uint32(size))
	if err != nil {
		return err
	}

	device, writeChar, notifyChar, err := connectToPeripheral()
	if err != nil {
		return err
	}
	defer device.Disconnect()

	notifyChar.EnableNotifications(func(buf []byte) {
		fmt.Println("Received delete image response:", string(buf))
	})

	return sendRequest(writeChar, deleteImage, payload)
}

func runList(c *cli.Context) error {
	device, writeChar, notifyChar, err := connectToPeripheral()
	if err != nil {
		return err
	}
	defer device.Disconnect()

	notifyChar.EnableNotifications(func(buf []byte) {
		fmt.Println("Received list image response:", string(buf))
	})

	err = writeHeader(writeChar, listImages)
	if err != nil {
		return err
	}
	time.Sleep(time.Second)
	return nil
}

func runGetFile(c *cli.Context) error {
	device, writeChar, notifyChar, err := connectToPeripheral()
	if err != nil {
		return err
	}
	defer device.Disconnect()

	notifyChar.EnableNotifications(func(buf []byte) {
		fmt.Println("Received get image response:", string(buf))
	})

	err = sendRequest(writeChar, getImage, []byte(c.Args()[0]))
	if err != nil {
		return err
	}
	time.Sleep(time.Second)
	return nil
}
