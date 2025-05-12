# BLE File Client (TinyGo)

This project is a command-line Bluetooth Low Energy (BLE) client implemented in Go, designed for use with [TinyGo](https://tinygo.org/). It interacts with a MicroPython-based BLE file server over a custom protocol.

The client supports several operations via BLE: health-check echo, file upload, file deletion, and listing files stored on the server.

---

# Usage

You can use the following CLI subcommands: Replace <BLE_ADDRESS> with the MAC address of your BLE peripheral.

## Echo (Health Check)
blecli echo --addr <BLE_ADDRESS>
## Upload a File
blecli upload --addr <BLE_ADDRESS> --file ./path/to/file.txt
## Delete a File
blecli delete --addr <BLE_ADDRESS> --md5 <16-byte MD5>
## List Files
blecli list --addr <BLE_ADDRESS>
## Convert one file into 7 color format
`blecli convert img <input filename>`
The command will do following tasks:
- resize input file to 800 x 480 size
- run FLOYDSTEINBERG algorithm to convert RGB to 7 color
- save dithered image to bmp file whose filename is <input filename>.bmp
- save raw dithered image data to one binary file whose filename is <input filename>.epa.
- to align with the epaper display, each pixel use 4 byte. The high 4 byte is for the 1st pixel, and low 4 byte is for the 2nd pixel, and so on. Epaper display app can load the content directly for display.
## Convert raw dithered image data to bmp format for manual verification
`blecli convert raw <input data filename>`
This command will read the raw data whose suffix is ".epa", and save it into bmp format, so that people can read it directly.

7 color palette
```
	color.RGBA{0, 0, 0, 255},       // Black
	color.RGBA{255, 255, 255, 255}, // White
	color.RGBA{0, 255, 0, 255},     // Green
	color.RGBA{0, 0, 255, 255},     // Blue
	color.RGBA{255, 0, 0, 255},     // Red
	color.RGBA{255, 255, 0, 255},   // Yellow
	color.RGBA{255, 128, 0, 255},   // Orange
```


# BLE Protocol

This BLE protocol is designed for communication between a BLE client and a MicroPython BLE peripheral acting as a file server. It uses two characteristics:

## UUIDs
- Write Characteristic UUID: 6e40
- Notify Characteristic UUID: 6e41
- Service UUID: 1234

## Format
Every message starts with a method byte, which defines the operation. The rest of the payload varies based on the method.


| Method|Value|Description|
| -------- | ------- |  ------- |
| 0x00|ECHO|Echo back received data|
| 0x01|UPLOAD|Send a file|
| 0x02|DELETE|Delete a file|
| 0x03|LIST|List files|


### Method: 0x00 (ECHO)
[1 byte method = 0x00][payload...]
Server echoes back [payload...] via notification.
### Method: 0x01 (UPLOAD)
[1 byte method = 0x01]
[4 bytes: file size (big endian)]
[16 bytes: MD5 of file (used as filename)]
[file contents...]
Server stores the file using MD5 hash as filename.
### Method: 0x02 (DELETE)
[1 byte method = 0x02]
[16 bytes: MD5 of file to delete]
### Method: 0x03 (LIST)
[1 byte method = 0x03]
Server replies with file list formatted as:
<filename1>.<size1>;<filename2>.<size2>;...
Each filename is a 16-byte MD5 string.

# Note

- MTU size can limit transmission chunk size. The client introduces a 1-second delay after each write.
- This repo is compatible with TinyGo.
- All files are stored on the server using their MD5 hash as filenames.

# Regenerating Code in the Future

If you'd like me to regenerate the Go CLI tool or MicroPython server again in the future:

Refer to this README's protocol section.
Mention that the BLE server runs MicroPython using aioble, and the client is a Go TinyGo app using github.com/urfave/cli.
Provide any new method IDs or protocol changes you'd like to add.
This will allow me to consistently regenerate the correct CLI and BLE code for your setup.