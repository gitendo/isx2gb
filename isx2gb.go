package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

const ver = 1.00

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
type record struct {
	bank     byte
	offset   uint16
	lenght   uint16
	overflow bool
}

// present rom, sram, ram data layout, size summary and overflow check
func areaSummary(area []record) {

	if len(area) > 0 {
		// sort by bank and offset
		area = sortRecords(area)

		bank := area[0].bank
		overflow := bool(false)
		total := uint16(0)
		prev := uint16(0)
		next := uint16(0)

		fmt.Printf("\nBank $%02x:\n", bank)

		for _, entry := range area {

			if entry.bank > bank {
				fmt.Printf("\t\t\t\t-----\n\t\t\t\t%5d bytes\n", total)
				total = 0
				bank = entry.bank
				fmt.Printf("\nBank $%02x:\n", bank)
			}

			next = entry.offset + entry.lenght
			fmt.Printf("\t\t$%04X - $%04X    %4d", entry.offset, next-1, entry.lenght)

			if entry.overflow {
				overflow = true
				fmt.Println(" - overflow!")
			} else {
				fmt.Println("")
			}
			// gets buggy w/out sorting
			if prev < entry.offset {
				total += entry.lenght
			} else {
				// exclude overlapped bytes
				if next > prev {
					total += (next - prev)
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

// scan isx records, create rom map and count used banks
func parseISXData(data []byte, size int) int {

	rom := []record{}
	ram := []record{}
	sram := []record{}
	bogus := []record{}
	used := byte(0)
	i := 0

	for i < size {
		switch data[i] {
		case 0x01:
			if data[i+loBank] == 0x80 {
				fmt.Fprintln(os.Stderr, "Error: ROMs above 16Mbits are not supported yet")
				os.Exit(1)
			} else {
				if data[i+loBank] > used {
					used = data[i+loBank]
				}

				entry := record{bank: data[i+loBank], offset: binary.LittleEndian.Uint16(data[i+loOffset:]), lenght: binary.LittleEndian.Uint16(data[i+loLength:]), overflow: false}
				length := entry.offset + entry.lenght

				switch {
				case entry.offset >= 0x0000 && entry.offset < 0x8000:
					if length >= 0x8000 {
						entry.overflow = true
					}
					rom = append(rom, entry)
				case entry.offset >= 0xA000 && entry.offset < 0xC000:
					if length >= 0x2000 {
						entry.overflow = true
					}
					sram = append(sram, entry)
				case entry.offset >= 0xC000 && entry.offset < 0xE000:
					if length >= 0x2000 {
						entry.overflow = true
					}
					ram = append(ram, entry)
				default:
					bogus = append(bogus, entry)
				}

				i = i + loHeader + int(entry.lenght)
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

	areaSummary(rom)
	areaSummary(sram)
	areaSummary(ram)
	areaSummary(bogus)

	return int(used)
}

// fill rom image with code blocks
func copyISXBinary(data []byte, rom []byte, size int) {

	var address, i, length int
	for i < size {
		switch data[i] {
		case 0x01:
			if data[i+loBank] != 0x80 {
				address = (int(data[i+loBank]) * 16384) + int(binary.LittleEndian.Uint16(data[i+loOffset:])&0x3FFF)
				length = i + loHeader + int(binary.LittleEndian.Uint16(data[i+loLength:]))
				copy(rom[address:], data[i+loHeader:length])
				i = length
			} else {
				panic("ROMs above 16Mbits are not supported yet!")
			}

		// case 0x03:
		// case 0x04:
		// case 0x11:
		// case 0x13:
		// case 0x14:
		default:
			panic("Unknown record type!")
		}
	}
}

func main() {

	fmt.Printf("\nisx2gb v%.2f - Intelligent Systems eXecutable converter for Game Boy (Color)\n", ver)
	fmt.Println("Programmed by: tmk, email: tmk@tuta.io")
	fmt.Printf("Project page: https://github.com/gitendo/isx2gb/\n\n")

	flgSort := flag.Bool("s", false, "sort isx records by rom bank / offset")
	flag.Parse()
	//	flag.Usage = usage

	// print usage if no input
	if len(os.Args) == 1 {
		flag.PrintDefaults()
		//		os.Exit(1)
	}

	fmt.Println("flag:", *flgSort)

	// access FileInfo - get file name and size
	isx, err := os.Stat("lancelot.isx")
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

	// read isx file
	data, err := ioutil.ReadFile(fn)
	if err, ok := err.(*os.PathError); ok {
		fmt.Fprintln(os.Stderr, "Error: Unable to read file", err.Path)
		os.Exit(1)
	}

	// 2nd check
	hdr := string(data[:headerSize])
	if !strings.HasPrefix(hdr, "ISX ") {
		fmt.Fprintln(os.Stderr, "Error: Header not found, invalid file")
		os.Exit(1)
	}

	if strings.HasSuffix(fn, ".isx") {
		fn = strings.Replace(fn, ".isx", ".gb", 1)
	} else {
		fn += ".gb"
	}

	// fancy
	hdr = strings.Replace(hdr, "    ", "", 1)
	fmt.Println(fn, " - ", hdr)

	banks := parseISXData(data[32:], fs-headerSize)
	banks++
	/*
		rom := make([]byte, banks*16384)
		copyISXBinary(data[32:], rom[:], fs-headerSize)

		// write ROM file
		err = ioutil.WriteFile(fn, rom, 0644)
		if err, ok := err.(*os.PathError); ok {
			fmt.Fprintln(os.Stderr, "Error: Unable to write file", err.Path)
			os.Exit(1)
		}
	*/
}
