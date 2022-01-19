package main

import (
	"fmt"
	"os"
	"path"
)

func read(f *os.File, cp int) ([]byte, int, error) {
	buffer := []byte{}
	buffer = make([]byte, cp)
	count, err := f.Read(buffer)
	if err != nil {
		return buffer, count, err
	}
	return buffer, count, err
}

func exitOnError(err error) {
	if err != nil {
		fmt.Printf("Error! %s\n", err.Error())
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Error! No filename given\n")
		os.Exit(1)
	}
	for a := range os.Args[1:] {
		filename := os.Args[a+1]
		file, err := os.Open(filename)
		if err != nil {
			fmt.Printf("Error opeing the file '%s'. %s\n", path.Base(filename), err.Error())
			os.Exit(1)
		}
		data, _, err := read(file, 2)
		exitOnError(err)
		if data[0] != 0xFF && data[1] != SOI {
			fmt.Printf("The file '%s' is not a valid JPEG file\n", path.Base(filename))
			os.Exit(1)
		}
		fmt.Printf("The file '%s' is a valid JPEG file\n", path.Base(filename))
		// TODO: Read all the remainig Markers!
		decodeJPEG(file)
	}
}

func decodeAPPN(f *os.File, header *Header, bt byte) {
	fmt.Printf("***** Decoding APPN Marker : 0xFF%X *****\n", bt)
	data, _, _ := read(f, 2)
	length := int((data[0] << 8) + data[1])
	read(f, length-2)
}

func padInt(a int) string {
	str := fmt.Sprintf("%d", a)
	rem := 3 - len(str)
	for k := 0; k < rem; k++ {
		str = " " + str
	}
	return str
}

func printHeader(header *Header) {
	fmt.Printf("***** DQT *****\n")
	for a := 0; a < 4; a++ {
		if header.quantizationTables[a].set {
			fmt.Printf("ID: %d\n", a)
			for b := 0; b < 64; b++ {
				if b%8 == 0 {
					fmt.Printf("\n")
				}
				fmt.Printf("%s ", padInt(int(header.quantizationTables[a].table[b])))
			}
			fmt.Printf("\n\n")
		}
	}
	fmt.Printf("***** SOF *****\n")
	fmt.Printf("FrameType: 0x%x\n", header.frameType)
	fmt.Printf("Width: %d\n", header.width)
	fmt.Printf("Height: %d\n", header.height)
	fmt.Printf("ColorComponents: %d\n", header.components)
	for a := 0; a < header.components; a++ {
		fmt.Printf("ComponentId: %d\n", a+1)
		fmt.Printf("Horizontal Sampling Factor: %d\n", header.colorComponents[a].hSamplingFactor)
		fmt.Printf("Vertical Sampling Factor: %d\n", header.colorComponents[a].vSamplingFactor)
		fmt.Printf("Quantization Table ID: %d\n", header.colorComponents[a].qTableId)
	}
}

var zigzag = []uint{
	0, 1, 8, 16, 9, 2, 3, 10,
	17, 24, 32, 25, 18, 11, 4, 5,
	12, 19, 26, 33, 40, 48, 41, 34,
	27, 20, 13, 6, 7, 14, 21, 28,
	35, 42, 49, 56, 57, 50, 43, 36,
	29, 22, 15, 23, 30, 37, 44, 51,
	58, 59, 52, 45, 38, 31, 39, 46,
	53, 60, 61, 54, 47, 55, 62, 63,
}

func decodeQT(f *os.File, header *Header, bt byte) {
	fmt.Printf("***** Decoding Quantization Table : 0xFF%X *****\n", bt)
	data, _, _ := read(f, 2)
	length := int((data[0] << 8) + data[1])
	length -= 2
	for {
		if length <= 0 {
			break
		}
		data, _, _ = read(f, 1)
		length -= 1
		tableId := data[0] & 0x0F

		// The table id can only be 0,1,2,3
		if tableId > 3 {
			fmt.Printf("Error! Invalid TableID --> %d\n", int(tableId))
			os.Exit(1)
		}
		header.quantizationTables[tableId].set = true
		isEightBit := (data[0] >> 4) == 0
		if !isEightBit {
			for a := 0; a < 64; a++ {
				data, _, _ := read(f, 2)
				header.quantizationTables[tableId].table[zigzag[a]] = (data[0] << 8) + data[1]
			}
			length -= 128
		} else {
			for a := 0; a < 64; a++ {
				data, _, _ := read(f, 1)
				header.quantizationTables[tableId].table[zigzag[a]] = data[0]
			}
			length -= 64
		}
	}
	if length != 0 {
		fmt.Printf("Error! Invalid DQT\n")
		os.Exit(1)
	}
}

func decodeSOF(f *os.File, header *Header, bt byte) {
	fmt.Printf("***** Decoding Start Of Frame : 0xFF%X *****\n", bt)
	header.frameType = bt
	if header.components != 0 {
		fmt.Printf("Error! Found more than one SOF0 Markers!\n")
		header.valid = false
		os.Exit(1)
	}
	data, _, _ := read(f, 2)
	length := int((data[0] << 8) + data[1])
	data, _, _ = read(f, 1)
	if data[0] != 8 {
		fmt.Printf("Error! Invalid precision %d\n", int(data[0]))
		header.valid = false
		os.Exit(1)
	}
	data, _, _ = read(f, 2)
	header.height = (int(data[0]) << 8) + int(data[1])
	data, _, _ = read(f, 2)
	header.width = (int(data[0]) << 8) + int(data[1])
	if header.width == 0 || header.height == 0 {
		fmt.Printf("Error! Ivalid dimensions. Width: %d, Height: %d\n", header.width, header.height)
		os.Exit(1)
	}
	data, _, _ = read(f, 1)
	header.components = int(data[0])
	if header.components == 4 {
		fmt.Printf("Error! CMYK mode not supported\n")
		header.valid = false
		os.Exit(1)
	}
	if header.components == 0 {
		fmt.Printf("Number of components must not be 0\n")
		os.Exit(1)
	}
	for a := 0; a < header.components; a++ {
		data, _, _ = read(f, 1)
		compId := int(data[0])
		// YIQ color mode uses Id 4 and Id 5
		if compId == 4 || compId == 5 {
			fmt.Printf("Error! YIQ Color mode not supported!\n")
			header.valid = false
			os.Exit(1)
		}
		// componentId should be 1,2,3
		if compId == 0 || compId > 3 {
			fmt.Printf("Error! Invalid compoenentId: %d\n", compId)
			os.Exit(1)
		}
		comp := &header.colorComponents[compId-1]
		if comp.used {
			fmt.Printf("Error! Duplicate ColorMode Id: %d\n", compId)
			header.valid = false
			os.Exit(1)
		}
		comp.used = true
		data, _, _ = read(f, 1)
		samplingFactor := data[0]
		// The sampling factor is divided into the upper and lower nibble for hSF and vSF
		comp.hSamplingFactor = int(samplingFactor >> 4)
		comp.vSamplingFactor = int(samplingFactor & 0x0F)
		if comp.hSamplingFactor != 1 || comp.vSamplingFactor != 1 {
			// TODO: Better Error Message
			header.valid = false
			fmt.Printf("Unsupported Samling Factor\n")
		}
		data, _, _ = read(f, 1)
		comp.qTableId = int(data[0])
		if comp.qTableId > 3 {
			fmt.Printf("Error! Invalid Quantization Table Id --> %d\n", comp.qTableId)
			header.valid = false
			os.Exit(1)
		}
	}
	if (length - 8 - (3 * header.components)) != 0 {
		fmt.Printf("Error! SOF Inavalid\n")
		header.valid = false
		os.Exit(1)
	}
}

func decodeJPEG(f *os.File) {
	header := &Header{
		valid: true,
	}
	data, _, _ := read(f, 2)

	for {
		if data[0] != 0xFF {
			fmt.Printf("Error! Expected a Marker but found --> %X\n", data[0])
			break
		}
		if data[1] >= APP0 && data[1] <= APP15 {
			decodeAPPN(f, header, data[1])
		}
		if data[1] == DQT {
			decodeQT(f, header, data[1])
		}
		if data[1] == SOF0 {
			decodeSOF(f, header, data[1])
		}
		data, _, _ = read(f, 2)
	}
	printHeader(header)
}

// The markers
const (
	SOI  = 0xD8
	SOF0 = 0xC0
	SOF2 = 0xC2
	DHT  = 0xC4
	DQT  = 0xDB
	DRI  = 0xDD
	SOS  = 0xDA
	/** Restart **/
	RST0 = 0xD0
	RST1 = 0xD1
	RST2 = 0xD2
	RST3 = 0xD3
	RST4 = 0xD4
	RST5 = 0xD5
	RST6 = 0xD6
	RST7 = 0xD7
	/** APP(n) Marker **/
	APP0  = 0xE0
	APP1  = 0xE1
	APP2  = 0xE2
	APP3  = 0xE3
	APP4  = 0xE4
	APP5  = 0xE5
	APP6  = 0xE6
	APP7  = 0xE7
	APP8  = 0xE8
	APP9  = 0xE9
	APP10 = 0xEA
	APP11 = 0xEB
	APP12 = 0xEC
	APP13 = 0xED
	APP14 = 0xEE
	APP15 = 0xEF
	/********/
	COM = 0xFE
	EOI = 0xD9
)

type QuantizationTable struct {
	table [64]byte
	set   bool // Is the table populated
}

type Header struct {
	frameType          byte // Baseline or progressive
	width              int
	height             int
	components         int
	valid              bool // Is the header valid
	quantizationTables [4]QuantizationTable
	colorComponents    [3]ColorComponent
}

type ColorComponent struct {
	hSamplingFactor int
	vSamplingFactor int
	qTableId        int
	used            bool
}
