package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

const loHeader = 6
const loBank = 1
const loOffset = 2
const loLength = 4

const hiHeader = 7
const hiBank = 2
const hiOffset = 3
const hiLength = 5

const headerSize = 32

type record struct {
	bank     byte
	offset   uint16
	lenght   uint16
	overflow bool
}

type records struct {
	rom   *[]record
	sram  *[]record
	ram   *[]record
	bogus *[]record
}

//
func printSummary(area []record) {

	if len(area) > 0 {
		area = sortRecords(area)

		bank := area[0].bank
		fmt.Printf("\nBank $%02x:\n", bank)
		total := uint16(0)
		overflow := bool(false)

		for _, v := range area {
			if v.bank > bank {
				bank = v.bank
				fmt.Printf("\t\t\t\t-----\n\t\t\t\t%5d bytes\n", total)
				total = 0
				fmt.Printf("\nBank $%02x:\n", bank)
			}

			fmt.Printf("\t\t$%04X - $%04X    %4d", v.offset, v.offset+v.lenght, v.lenght)
			if v.overflow {
				fmt.Println(" - overflow!")
			} else {
				fmt.Println("")
			}
			//	total += v.lenght

		}

		fmt.Printf("\t\t\t\t-----\n\t\t\t\t%5d bytes\n", total)

		if overflow {
			fmt.Fprintf(os.Stderr, "\nError: (!) data overflow detected\n")
			os.Exit(1)
		}

	}
}

// sort isx records (rom, ram, sram) according to bank and offset
func sortRecords(area []record) []record {

	sort.Slice(area, func(i, j int) bool {
		if area[i].bank < area[j].bank {
			return true
		}
		if area[i].bank > area[j].bank {
			return false
		}
		return area[i].offset < area[j].offset
	})
	return area
}

// scan isx records, create ROM map and count used banks
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

				if entry.offset >= 0x0000 && entry.offset < 0x8000 {
					if (entry.offset + entry.lenght) >= 0x8000 {
						entry.overflow = true
					}
					rom = append(rom, entry)
				} else if entry.offset >= 0xA000 && entry.offset < 0xC000 {
					if (entry.offset + entry.lenght) >= 0x2000 {
						entry.overflow = true
					}
					sram = append(sram, entry)
				} else if entry.offset >= 0xC000 && entry.offset < 0xE000 {
					if (entry.offset + entry.lenght) >= 0x2000 {
						entry.overflow = true
					}
					ram = append(ram, entry)
				} else {
					bogus = append(bogus, entry)
				}

				i += int(entry.lenght)
				i += loHeader
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

	printSummary(rom)
	printSummary(sram)
	printSummary(ram)
	//, &sram, &ram, &bogus}

	return int(used)
}

// fill ROM image with code blocks
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
