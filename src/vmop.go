package main

import (
	"os"
	"strings"
	"path/filepath"
	"quark/vm"
	"fmt"
	"encoding/binary"
)

// BuildGluon: merges packages and project sources into one canonical Gluon blob
func BuildGluon(projectDir string) error {
	// master containers
	var finishedCode []byte
	var finishedConsts []interface{}
	processed := map[string]bool{} // use checksum string map for dedupe

	// local opcode constants - MUST match your VM's opcodes
	const (
		OpHalt         = 0x00
		OpLoadConst    = 0x01 // followed by u16 const index
		OpStoreLocal   = 0x02 // followed by u16 local index
		OpLoadLocal    = 0x03 // followed by u16 local index
		OpAdd          = 0x04
		OpSub          = 0x05
		OpMul          = 0x06
		OpDiv          = 0x07
		OpCallBuiltin  = 0x08 // followed by u8 argc
		OpPop          = 0x09
		OpJump         = 0x0A // followed by u16 addr
		OpJumpIfFalse  = 0x0B // followed by u16 addr
	)

	// helper: remap all LOAD_CONST u16 indices in code by adding offset
	remapLoadConstIndices := func(code []byte, offset uint16) ([]byte, error) {
		out := make([]byte, len(code))
		copy(out, code)

		i := 0
		for i < len(out) {
			op := out[i]
			i++
			switch op {
			case OpHalt:
				// no operands
			case OpLoadConst:
				if i+2 > len(out) {
					return nil, fmt.Errorf("malformed code while reading LOAD_CONST operand")
				}
				idx := binary.LittleEndian.Uint16(out[i : i+2])
				newIdx := idx + offset
				binary.LittleEndian.PutUint16(out[i:i+2], newIdx)
				i += 2
			case OpStoreLocal, OpLoadLocal, OpJump, OpJumpIfFalse:
				// u16 operand
				if i+2 > len(out) {
					return nil, fmt.Errorf("malformed code while reading u16 operand for op %d", op)
				}
				i += 2
			case OpCallBuiltin:
				// single u8 operand (argc)
				if i+1 > len(out) {
					return nil, fmt.Errorf("malformed code while reading CALL operand")
				}
				i += 1
			case OpAdd, OpSub, OpMul, OpDiv, OpPop:
				// no inline operands
			default:
				return nil, fmt.Errorf("unknown opcode %d while remapping", op)
			}
		}
		return out, nil
	}

	// helper to process a single blob (source.glue or compiled blob)
	processBlob := func(blob []byte) error {
		// dedupe by checksum of the raw blob
		chk := Checksum(blob)
		if processed[chk] {
			return nil
		}
		processed[chk] = true

		// Deserialize - assumes vm exposes a DeserializeBytecode(blob) -> (code, consts, err)
		code, consts, err := vm.DeserializeBytecode(blob)
		if err != nil {
			return fmt.Errorf("deserialize failed: %w", err)
		}

		// offset is current number of finishedConsts
		offset := uint16(len(finishedConsts))

		// remap load-const operands in code
		remappedCode, err := remapLoadConstIndices(code, offset)
		if err != nil {
			return fmt.Errorf("remap error: %w", err)
		}

		// append consts into finishedConsts
		for _, c := range consts {
			finishedConsts = append(finishedConsts, c)
		}
		// append remapped code
		finishedCode = append(finishedCode, remappedCode...)
		return nil
	}

	// 1) process Gluons in pkgs/
	pkgsDir := filepath.Join(projectDir, "pkgs")
	if _, err := os.Stat(pkgsDir); err == nil {
		if err := filepath.Walk(pkgsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".gluon") {
				LoadGluon(path) // mounts to quark--gluon--mount
				// read source.glue inside mount
				gluonBlobPath := filepath.Join("quark--gluon--mount", "source.glue")
				if _, err := os.Stat(gluonBlobPath); err == nil {
					blob, err := os.ReadFile(gluonBlobPath)
					if err != nil {
						return err
					}
					if err := processBlob(blob); err != nil {
						return err
					}
				} else {
					// no source.glue found - maybe packaged differently; skip or warn
					Log("warning: gluon had no source.glue: " + path)
				}
				// cleanup mount
				os.RemoveAll("quark--gluon--mount")
			}
			return nil
		}); err != nil {
			return err
		}
	}

	// 2) compile each .quark source file into its blob and process the blob
	if err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".quark") {
			srcBytes, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			chk := Checksum(srcBytes)
			if processed[chk] {
				return nil
			}
			blob, err := vm.CompileSourceToBlob(string(srcBytes))
			if err != nil {
				return err
			}
			if err := processBlob(blob); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// 3) append a single HALT at the end (make sure compiler removed per-file HALT)
	finishedCode = append(finishedCode, byte(OpHalt))

	// 4) serialize combined consts + combined code into final blob
	finalBlob, err := vm.SerializeBytecode(finishedCode, finishedConsts)
	if err != nil {
		return fmt.Errorf("serialize failed: %w", err)
	}

	// 5) hand to MakeGluon (MakeGluon must accept []byte)
	return MakeGluon(finalBlob)
}

