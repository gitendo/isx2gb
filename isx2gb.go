package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const ver = 1.00

const bankSize = 0x4000

// ROM header stuff
const logoStart = 0x0104
const logoEnd = 0x0133
const titleStart = 0x0134
const cgbFlag = 0x0143
const headerCRC = 0x014D
const globalCRC = 0x014E
const logoCRC = 0x153807cd

// roms up to 128Mbit
const loHeader = 6
const loBank = 1
const loOffset = 2
const loLength = 4

// roms over 128Mbit
const hiHeader = 7
const hiBank = 2
const hiOffset = 3
const hiLength = 5

// isx string
const headerSize = 32

// used to present data layout
// status: -1 overflow, 0 normal, 1 spanned
type record struct {
	bank   byte
	offset uint16
	ptr    int
	length uint16
	status byte
}

// options
var optFil *bool
var optPad *bool
var optRec *bool

//var optSym *bool

// present rom, sram, ram data layout, size summary and overflow check
func areaDetails(area []record, name string) {

	if len(area) > 0 {
		// sort by bank and offset
		area = sortRecords(area)
		bank := area[0].bank
		overflow := bool(false)
		total := uint16(0)
		prev := uint16(0)
		next := uint16(0)
		flg := bool(true)

		fmt.Printf("%s Bank $%02x:\n", name, bank)

		for _, entry := range area {

			if entry.bank > bank {
				fmt.Printf("\t\t\t\t-----\n\t\t\t\t%5d bytes\n", total)
				total = 0
				bank = entry.bank
				fmt.Printf("\n%s Bank $%02x:\n", name, bank)
			}

			next = entry.offset + entry.length

			switch entry.status {
			case 0x01:
				if flg {
					fmt.Printf("\t\t$%04X -   >      %4d\n", entry.offset, entry.length)
					flg = false
				} else {
					fmt.Printf("\t\t  >   - $%04X    %4d\n", next-1, entry.length)
					flg = true
				}
			case 0xFF:
				fmt.Printf("\t\t$%04X - $%04X    %4d   !\n", entry.offset, next-1, entry.length)
				overflow = true
			default:
				fmt.Printf("\t\t$%04X - $%04X    %4d\n", entry.offset, next-1, entry.length)
			}

			// gets buggy w/out sorting
			if prev < entry.offset {
				total += entry.length
			} else {
				// exclude overlapped bytes
				if next > prev {
					total += next - prev
				}
			}
			prev = next
		}

		fmt.Printf("\t\t\t\t-----\n\t\t\t\t%5d bytes\n", total)

		if overflow {
			fmt.Fprintf(os.Stderr, "\nError: data overflow detected\n")
			os.Exit(1)
		}

	}
}

// sort areas (rom, ram, sram) according to bank and offset
func sortRecords(area []record) []record {

	sort.Slice(area, func(i, j int) bool {
		switch {

		case area[i].bank != area[j].bank:
			return area[i].bank < area[j].bank

		default:
			return area[i].offset < area[j].offset
		}
		/*
			if area[i].bank < area[j].bank {
				return true
			}
			if area[i].bank > area[j].bank {
				return false
			}
			return area[i].offset < area[j].offset
		*/
	})
	return area
}

// scan isx records, create areas and count used banks
func parseISXData(f *os.File, fn string, fs int) {

	rom := []record{}
	ram := []record{}
	sram := []record{}
	bogus := []record{}
	used := byte(0)
	i := 0

	// read isx records
	data := make([]byte, fs)
	_, err := f.Read(data)
	f.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Unable to read from", fn)
		os.Exit(1)
	}

	// scan and gather required info
	for i < fs {
		switch data[i] {
		case 0x01:
			if data[i+loBank] == 0x80 {
				fmt.Fprintln(os.Stderr, "Error: ROMs above 16Mbits are not supported yet")
				os.Exit(1)
			} else {
				if data[i+loBank] > used {
					used = data[i+loBank]
				}

				entry := record{bank: data[i+loBank], offset: binary.LittleEndian.Uint16(data[i+loOffset:]), ptr: i + loHeader, length: binary.LittleEndian.Uint16(data[i+loLength:]), status: 0}
				length := entry.offset + entry.length
				// for loop control
				i = i + loHeader + int(entry.length)

				switch {
				// rom area
				case entry.offset >= 0x0000 && entry.offset < 0x8000:

					if length >= 0x8000 {
						// overflow
						entry.status--
					}
					// allow data overflow in bank 0
					if entry.bank == 0 {
						// data migt be in bank 0 or
						if entry.offset < bankSize {
							// might be spanned across bank 0 and 1
							if length >= bankSize {
								entry.status++
								entry.length = bankSize - entry.offset
								rom = append(rom, entry)
								entry.bank++
								used = 1
								entry.offset = 0x4000
								entry.ptr += int(entry.length)
								entry.length = length - bankSize
							}
							// data is located in bank 1
						} else {
							entry.bank++
							used = 1
						}
					}
					rom = append(rom, entry)
				// sram area
				case entry.offset >= 0xA000 && entry.offset < 0xC000:
					if length > 0xBFFF {
						// overflow
						entry.status--
					}
					sram = append(sram, entry)
				// ram area
				case entry.offset >= 0xC000 && entry.offset < 0xE000:
					if length > 0xDFFF {
						// overflow
						entry.status--
					}
					ram = append(ram, entry)
				// invalid records
				default:
					bogus = append(bogus, entry)
				}

			}

		// case 0x03:
		// case 0x04:
		// case 0x11:
		// case 0x13:
		// case 0x14:
		default:
			fmt.Fprintf(os.Stderr, "Error: Unknown record type (%X : %X)", i, data[i])
			os.Exit(1)
		}
	}
	// bank 0 is still one used bank ;)
	used++

	if *optRec {
		areaDetails(rom, "ROM")
		areaDetails(sram, "SRAM")
		areaDetails(ram, "RAM")
		areaDetails(bogus, "???")
		areas := [][]record{rom, sram, ram}
		dumpISXRecords(areas, data, fn)
	} else {
		areaDetails(rom, "ROM")
		makeROM(rom, data, int(used), fn)
	}
}

// extract single record and save it as standalone file
func dumpISXRecords(areas [][]record, data []byte, fn string) {

	var length int
	var nn string

	for _, area := range areas {
		for _, entry := range area {
			length = int(entry.ptr) + int(entry.length)
			// not your average colon, it's U+A789
			nn = fmt.Sprintf("%s_%02X꞉%04X.bin", fn, entry.bank, entry.offset)
			// write file
			err := ioutil.WriteFile(nn, data[entry.ptr:length], 0644)
			if err, ok := err.(*os.PathError); ok {
				fmt.Fprintln(os.Stderr, "Error: Unable to write file", err.Path)
				os.Exit(1)
			}
			fmt.Println(nn)
		}
	}

}

// create rom image, update header checksums and save it
func makeROM(area []record, data []byte, banks int, fn string) {

	var offset, length int

	// ROM padding
	if *optPad {
		banks--
		banks |= banks >> 1
		banks |= banks >> 2
		banks |= banks >> 4
		banks++
	}

	rom := make([]byte, banks*bankSize)

	// fill ROM with 0xFF values
	if *optFil {
		rom[0] = 0xFF
		for i := 1; i < len(rom); i *= 2 {
			copy(rom[i:], rom[:i])
		}
	}

	for _, entry := range area {
		offset = int(entry.offset)
		length = int(entry.ptr) + int(entry.length)
		copy(rom[offset:], data[entry.ptr:length])
	}

	// fix checksums if there's valid Nintendo logo
	crc32q := crc32.MakeTable(0xD5828281)
	crc := crc32.Checksum(rom[logoStart:logoEnd], crc32q)
	if crc == logoCRC {

		// calculate header checksum
		crc = 0
		for i := titleStart; i < headerCRC; i++ {
			crc = crc - uint32(rom[i]) - 1
		}
		rom[headerCRC] = byte(crc)

		// foolproof
		crc = 0
		binary.BigEndian.PutUint16(rom[globalCRC:], uint16(crc))

		// calculate global checksum
		for i := 0; i < len(rom); i++ {
			crc += uint32(rom[i])
		}
		binary.BigEndian.PutUint16(rom[globalCRC:], uint16(crc))
	}

	// hardware check
	if rom[cgbFlag] == 0xC0 {
		fn += ".gbc"
	} else {
		fn += ".gb"
	}

	// write ROM file
	err := ioutil.WriteFile(fn, rom, 0644)
	if err, ok := err.(*os.PathError); ok {
		fmt.Fprintln(os.Stderr, "Error: Unable to write file", err.Path)
		os.Exit(1)
	}
}

func main() {

	fmt.Printf("\nisx2gb v%.2f - Intelligent Systems eXecutable converter for Game Boy (Color)\n", ver)
	fmt.Println("Programmed by: tmk, email: tmk@tuta.io")
	fmt.Printf("Project page: https://github.com/gitendo/isx2gb/\n\n")

	// define program options
	optFil = flag.Bool("f", false, "switch ROM filling pattern to 0xFF")
	optPad = flag.Bool("p", false, "round up ROM size to the next highest power of 2")
	optRec = flag.Bool("r", false, "save isx records separately")
	//	optSym = flag.Bool("s", false, "create symbol file")

	flag.Usage = func() {
		fmt.Printf("Usage:\t%s [options] file[.isx]\n\n", filepath.Base(os.Args[0]))
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	flag.Parse()

	args := flag.Args()
	// print usage if no input
	if len(args) != 1 {
		flag.Usage()
	}

	// access FileInfo - get file name and size
	isx, err := os.Stat(args[0])
	if err, ok := err.(*os.PathError); ok {
		fmt.Fprintln(os.Stderr, "Error: Unable to access file", err.Path)
		os.Exit(1)
	}

	fn := isx.Name()
	fs := int(isx.Size())

	// 1st check
	if fs <= headerSize {
		fmt.Fprintln(os.Stderr, "Error: Dubious file size, probably invalid")
		os.Exit(1)
	}

	// open file
	f, err := os.Open(fn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Unable to access", fn)
		os.Exit(1)
	}

	// read header
	header := make([]byte, 32)
	_, err = f.Read(header)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Unable to read from", fn)
		f.Close()
		os.Exit(1)
	}

	// 2nd check
	if !strings.HasPrefix(string(header), "ISX ") {
		fmt.Fprintln(os.Stderr, "Error: Header not found, invalid file")
		f.Close()
		os.Exit(1)
	}

	// fancy
	fmt.Printf("%s : %s\n\n", fn, strings.Replace(string(header), "    ", "", 1))
	fn = strings.TrimSuffix(fn, ".isx")

	parseISXData(f, fn, fs-headerSize)
}
