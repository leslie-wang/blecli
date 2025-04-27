package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

func main() {
	app := cli.NewApp()
	app.Usage = "BLE client tools"

	app.Commands = []cli.Command{
		{
			Name:    "scan",
			Aliases: []string{"d"},
			Action:  scan,
		},
		{
			Name:    "list-photos",
			Aliases: []string{"lp"},
			Action:  listPhotos,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func listPhotos(ctx *cli.Context) error {
	return nil
}

func scan(ctx *cli.Context) error {
	// Enable BLE interface.
	err := adapter.Enable()
	if err != nil {
		return err
	}

	// Start scanning.
	var deviceAddress bluetooth.Address
	found := false
	println("scanning...")
	err = adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		if device.LocalName() != "pico2w_ble" {
			return
		}
		fmt.Printf("found device: %s, RSSI: %d, %s\n", device.Address.String(), device.RSSI, device.LocalName())
		deviceAddress = device.Address
		found = true
		adapter.StopScan()
	})
	if err != nil {
		return err
	}

	if !found {
		return errors.New("PicoServer not found")
	}

	fmt.Println("Connecting to PicoServer...")
	device, err := adapter.Connect(deviceAddress, bluetooth.ConnectionParams{})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer device.Disconnect()

	// Discover services and characteristics
	// # On the Pico W (MicroPython side)
	// ble_svc_uuid = bluetooth.UUID(0x181A)         # Environmental Sensing Service
	// ble_characteristic_uuid = bluetooth.UUID(0x2A6E)  # Temperature Characteristic

	serviceUUID := baseUUID(0x181A)
	writeUUID := baseUUID(0x2A6E).String()
	notifyUUID := baseUUID(0x2A1C).String()

	fmt.Println("Discovering Services PicoServer...")
	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		return err
	}
	if len(services) == 0 {
		return errors.New("service not found")
	}

	fmt.Printf("Discovered Services: %+v\n", services)

	// Discover characteristics
	fmt.Println("Discovering characteristics PicoServer...")

	chars, err := services[0].DiscoverCharacteristics(nil)
	if err != nil {
		return errors.Wrapf(err, "Could not find notify characteristic")
	}

	var notifyChar, writeChar *bluetooth.DeviceCharacteristic
	for _, char := range chars {
		switch char.UUID().String() {
		case writeUUID:
			writeChar = &char
		case notifyUUID:
			notifyChar = &char
		}
	}
	if notifyChar == nil {
		return errors.New("unable to discover notification characteristics")
	} else if writeChar == nil {
		return errors.New("unable to discover write characteristics")
	}

	fmt.Printf("Discovered Notify Characteristics: %+v\n", notifyChar)

	// Enable notifications before sending the request
	notifyChar.EnableNotifications(func(buf []byte) {
		fmt.Println("Received response:", string(buf))
	})

	time.Sleep(1 * time.Second)

	fmt.Printf("Discovered Write Characteristics: %+v\n", writeChar)

	// Send "Hello, world"
	fmt.Println("send hello world")
	message := []byte("hello")
	_, err = writeChar.WriteWithoutResponse([]byte(message))
	if err != nil {
		log.Fatalf("Failed to write: %v", err)
	}
	fmt.Printf("Message (%s) sent successfully.\n", message)
	time.Sleep(time.Minute)
	return nil
}

func baseUUID(short uint16) bluetooth.UUID {
	var b = [16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x80, 0x00, 0x00, 0x80, 0x5F, 0x9B, 0x34, 0xFB}
	b[2] = byte(short >> 8)
	b[3] = byte(short & 0xFF)
	return bluetooth.NewUUID(b)
}
