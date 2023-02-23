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

// version
const ver = 1.03

// options
var optDump *bool
var optFill *bool
var optPatch *bool
var optRound *bool
var optSym *bool

// rom header stuff
const logoStart = 0x0104
const logoEnd = 0x0133
const titleStart = 0x0134
const cgbFlag = 0x0143
const romSize = 0x148
const headerCRC = 0x014D
const globalCRC = 0x014E
const logoCRC = 0x153807cd

const bankSize = 0x4000

// isx string
const headerSize = 32

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

// used to create .sym file
type symbol struct {
	bank   byte
	offset uint16
	name   string
}

// used to present data layout
// status: -1 overflow, 0 normal, 1 spanned from, 2 spanned to
type record struct {
	bank   byte
	offset uint16
	ptr    int
	length uint16
	status byte
}

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

		fmt.Printf("%s Bank $%02x:\n", name, bank)

		for _, entry := range area {

			if entry.bank > bank {
				fmt.Printf("\t\t\t\t -----\n\t\t\t\t %5d bytes\n", total)
				total = 0
				prev = 0
				bank = entry.bank
				fmt.Printf("\n%s Bank $%02x:\n", name, bank)
			}

			next = entry.offset + entry.length
			// due to sorting, prev can't be > entry.offset
			if prev < entry.offset {
				total += entry.length
				// prev == entry.offset
			} else {
				// exclude overlapping bytes
				if next > prev {
					total += next - prev
				}
			}
			prev = next

			if entry.bank > 0 {
				entry.offset |= bankSize
			}

			switch entry.status {
			case 0x01:
				fmt.Printf("\t\t$%04X -   >      %4d\n", entry.offset, entry.length)
			case 0x02:
				fmt.Printf("\t\t  >   - $%04X    %4d\n", entry.offset+entry.length-1, entry.length)
			case 0xFF:
				fmt.Printf("\t\t$%04X - $%04X    %4d   !\n", entry.offset, entry.offset+entry.length-1, entry.length)
				overflow = true
			default:
				fmt.Printf("\t\t$%04X - $%04X    %5d\n", entry.offset, entry.offset+entry.length-1, entry.length)
			}
		}

		fmt.Printf("\t\t\t\t -----\n\t\t\t\t %5d bytes\n\n", total)

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
	})
	return area
}

// sort symbols according to flag, bank and offset
func sortSymbols(area []symbol) []symbol {

	sort.Slice(area, func(i, j int) bool {
		switch {
		// skip flags other than 0x1000
		case area[i].bank != area[j].bank:
			return area[i].bank < area[j].bank

		default:
			return area[i].offset < area[j].offset
		}
	})
	return area
}

// fix rom header and global checksums
func updateCRC(rom []byte) {

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
}

// scan isx records, create areas and count used banks
func parseISX(args []string) {

	rom := []record{}
	ram := []record{}
	sram := []record{}
	bogus := []record{}
	sym := []symbol{}
	used := byte(0)
	i := 0

	// open isx file
	fn := args[0]
	f, err := os.Open(fn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Unable to access", fn)
		os.Exit(1)
	}

	// access file info to get file size
	isx, err := f.Stat()
	if err, ok := err.(*os.PathError); ok {
		fmt.Fprintln(os.Stderr, "Error: Unable to access FileInfo for", err.Path)
		os.Exit(1)
	}
	fs := int(isx.Size())

	// 1st check
	if fs <= headerSize {
		fmt.Fprintln(os.Stderr, "Error: Dubious file size, probably invalid")
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
	fmt.Printf("%s : %s\n\n", isx.Name(), strings.Replace(string(header), "    ", "", 1))

	// strip extension, file name will be reused later
	fn = strings.TrimSuffix(fn, ".isx")

	// read isx records
	fs -= headerSize
	data := make([]byte, fs)
	_, err = f.Read(data)
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
				// force relative offsets in banks 1 and above
				if entry.bank > 0 && entry.offset >= bankSize {
					entry.offset &= 0x3FFF
				}

				// for loop control
				i = i + loHeader + int(entry.length)

				switch {
				// rom area
				case entry.offset >= 0x0000 && entry.offset < 0x8000:
					// allow overflow for bank 0 only
					if entry.bank == 0 {
						// data starts in bank 0
						if entry.offset < bankSize {
							// it might be spanned across bank 0 and 1, split it into 2 records
							if (entry.offset + entry.length) >= bankSize {
								// part in bank 0
								length := entry.length
								entry.length = bankSize - entry.offset
								entry.status = 1
								rom = append(rom, entry)
								// part in bank 1
								entry.bank = 1
								entry.offset = 0
								entry.ptr += int(entry.length)
								entry.length = length - entry.length
								entry.status = 2
								used = 1
							}
						} else {
							// data is located in bank 1
							entry.bank++
							entry.offset &= 0x3FFF
							used = 1
						}
					} else {
						// for other banks overflow is not allowed
						if (entry.offset + entry.length) > bankSize {
							entry.status = 0xFF
						}
					}
					rom = append(rom, entry)
				// sram area
				case entry.offset >= 0xA000 && entry.offset < 0xC000:
					if (entry.offset + entry.length) > 0xBFFF {
						// overflow
						entry.status = 0xFF
					}
					sram = append(sram, entry)
				// ram area
				case entry.offset >= 0xC000 && entry.offset < 0xE000:
					if (entry.offset + entry.length) > 0xDFFF {
						// overflow
						entry.status = 0xFF
					}
					ram = append(ram, entry)
				// invalid records
				default:
					bogus = append(bogus, entry)
				}

			}

		// case 0x03:
		// 	break
		// case 0x04:
		// 	break
		// case 0x11:
		// 	range information
		case 0x13:
			i++
			// number of range entries
			j := int(binary.LittleEndian.Uint16(data[i:]))
			i += 2
			i += (j * 9)

		// 	symbol information
		case 0x14:
			i++
			// number of symbols
			j := binary.LittleEndian.Uint16(data[i:])
			i += 2

			for j > 0 {
				length := int(data[i])
				i++
				name := string(data[i : i+length])
				i += length
				flag := binary.LittleEndian.Uint16(data[i:])
				i += 2
				offset := binary.LittleEndian.Uint16(data[i:])
				i += 2
				bank := data[i]
				i += 2
				if flag == 0x1000 {
					sym = append(sym, symbol{bank: bank, offset: offset, name: name})
				}
				j--
			}
		// debug information
		case 0x20, 0x21, 0x22:
			i++
			i += int(binary.LittleEndian.Uint32(data[i:])) + 4

		default:
			fmt.Fprintf(os.Stderr, "Error: Unknown record type %X at %X found\n", data[i], i+headerSize)
			os.Exit(1)
		}
	}
	// bank 0 is still one used bank ;)
	used++

	// save records
	if *optDump {
		areaDetails(rom, "ROM")
		areaDetails(sram, "SRAM")
		areaDetails(ram, "RAM")
		areaDetails(bogus, "???")
		areas := [][]record{rom, sram, ram}
		dumpISX(areas, data, fn)
		// patch rom
	} else if *optPatch {
		patchROM(rom, data, args[1])
		// save rom
	} else {
		areaDetails(rom, "ROM")
		makeROM(rom, data, int(used), fn)
	}

	// create symbolic file for debugger
	if *optSym {
		makeSYM(sym, fn)
	}
}

// write isx records to files
func dumpISX(areas [][]record, data []byte, fn string) {

	var length int
	var rn string

	fmt.Println("Dumping...")
	// iterate isx records, write each record to file
	for _, area := range areas {
		for _, record := range area {
			length = int(record.ptr) + int(record.length)
			// not your average colon, it's U+A789
			rn = fmt.Sprintf("%s_%02X꞉%04X.bin", fn, record.bank, record.offset)
			// write file
			err := ioutil.WriteFile(rn, data[record.ptr:length], 0644)
			if err, ok := err.(*os.PathError); ok {
				fmt.Fprintln(os.Stderr, "Error: Unable to write file", err.Path)
				os.Exit(1)
			}
			fmt.Println(rn)
		}
	}
	fmt.Printf("\nDone!\n")
}

// apply isx records to existing rom file
func patchROM(area []record, data []byte, fn string) {

	var length int
	var ptr int64
	var str string

	// open rom file to be patched
	f, err := os.Open(fn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Unable to access", fn)
		os.Exit(1)
	}

	// access file info to get file size
	isx, err := f.Stat()
	if err, ok := err.(*os.PathError); ok {
		fmt.Fprintln(os.Stderr, "Error: Unable to access FileInfo for", err.Path)
		os.Exit(1)
	}
	fs := isx.Size()

	// read rom file
	rom := make([]byte, fs)
	_, err = f.Read(rom)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Unable to read from", fn)
		os.Exit(1)
	}
	f.Close()

	fmt.Println("Patching...")

	// iterate isx records, treat each record as patch
	for _, record := range area {
		// calculate record length
		length = int(record.ptr) + int(record.length)
		// calculate and set rom file pointer
		ptr = int64(record.bank)*bankSize + int64(record.offset)
		// make sure we're within the rom buffer
		if ptr > fs {
			fmt.Fprintf(os.Stderr, "Error: Patching over ROM boundary is not possible: 0x%08X\n", ptr)
			os.Exit(1)
		}
		// apply patch
		copy(rom[ptr:], data[record.ptr:length])
		// verbose output
		if record.length == 1 {
			str = fmt.Sprintf("0x%08X: %5d byte", ptr, record.length)
		} else {
			str = fmt.Sprintf("0x%08X: %5d bytes", ptr, record.length)
		}
		fmt.Println(str)
	}

	// don't spread roms with bad crc, it's lame
	updateCRC(rom)

	// tag filename
	ext := filepath.Ext(fn)
	tag := "-patched"
	fn = strings.Replace(fn, ext, tag+ext, 1)

	// write rom file
	err = ioutil.WriteFile(fn, rom, 0644)
	if err, ok := err.(*os.PathError); ok {
		fmt.Fprintln(os.Stderr, "Error: Unable to write file", err.Path)
		os.Exit(1)
	}
	fmt.Printf("\n%s has been created!\n", fn)
}

// create rom image, update header checksums and save it
func makeROM(area []record, data []byte, banks int, fn string) {

	var offset, length int

	// rom padding - could be fixed (romSize value from header might be > banks)
	if *optRound {
		if banks > 1 {
			banks--
		}
		banks |= banks >> 1
		banks |= banks >> 2
		banks |= banks >> 4
		banks++
	}

	rom := make([]byte, banks*bankSize)

	// fill rom with 0xFF values
	if *optFill {
		rom[0] = 0xFF
		for i := 1; i < len(rom); i *= 2 {
			copy(rom[i:], rom[:i])
		}
	}

	// fill rom with isx records
	for _, record := range area {
		offset = int(record.bank)*bankSize + int(record.offset)
		length = int(record.ptr) + int(record.length)
		copy(rom[offset:], data[record.ptr:length])
	}

	// don't spread roms with bad crc, it's lame
	updateCRC(rom)

	// hardware check
	if rom[cgbFlag] == 0xC0 {
		fn += ".gbc"
	} else {
		fn += ".gb"
	}

	// write rom file
	err := ioutil.WriteFile(fn, rom, 0644)
	if err, ok := err.(*os.PathError); ok {
		fmt.Fprintln(os.Stderr, "Error: Unable to write file", err.Path)
		os.Exit(1)
	}
}

// create symbols file
func makeSYM(symbols []symbol, fn string) {

	if len(symbols) > 0 {
		fn += ".sym"
		f, err := os.Create(fn)
		if err, ok := err.(*os.PathError); ok {
			fmt.Fprintln(os.Stderr, "Error: Unable to create file", err.Path)
			return
		}

		symbols = sortSymbols(symbols)

		for _, record := range symbols {
			fmt.Fprintf(f, "%02X:%04X %s\n", record.bank, record.offset, record.name)
			if err, ok := err.(*os.PathError); ok {
				fmt.Fprintln(os.Stderr, "Error: Unable to write to file", err.Path)
				f.Close()
				return
			}
		}
		f.Close()
	} else {
		fmt.Println("Warning: File doesn't contain any symbolic information, check your config")
	}
}

func main() {

	fmt.Printf("\nisx2gb v%.2f - Intelligent Systems eXecutable utility for Game Boy (Color)\n", ver)
	fmt.Println("Programmed by: tmk, email: tmk@tuta.io")
	fmt.Printf("Project page: https://github.com/gitendo/isx2gb/\n\n")

	// define command line options
	optDump = flag.Bool("d", false, "dump isx records into binary file(s)")
	optFill = flag.Bool("f", false, "switch ROM filling pattern from 0x00 to 0xFF")
	optPatch = flag.Bool("p", false, "patch supplied ROM file with ISX records")
	optRound = flag.Bool("r", false, "round up ROM size to the next highest power of 2")
	optSym = flag.Bool("s", false, "create symbolic file for debugger")

	flag.Usage = func() {
		fmt.Printf("Usage:\t%s [options] file.isx [romfile.gb]\n\n", filepath.Base(os.Args[0]))
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		os.Exit(1)
	}

	flag.Parse()

	args := flag.Args()

	// print usage if no input or misinput
	if len(args) < 1 || len(args) > 2 {
		flag.Usage()
	}
	if *optPatch == true && len(args) < 2 {
		flag.Usage()
	}

	// let's do it!
	parseISX(args)
}
