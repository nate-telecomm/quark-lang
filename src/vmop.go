package main

import (
	"os"
	"strings"
	"path/filepath"
	"quark/vm"
	"fmt"
	"encoding/binary"
)

func BuildGluon(projectDir string) error {
	var finishedCode []byte
	var finishedConsts []interface{}
	processed := map[string]bool{}
	remapLoadConstIndices := func(code []byte, offset uint16) ([]byte, error) {
		out := make([]byte, len(code))
		copy(out, code)

		i := 0
		for i < len(out) {
			op := out[i]
			i++
			switch op {
			case vm.OpHalt:
				// hello debuggers!
			case vm.OpLoadConst:
				if i+2 > len(out) {
					return nil, fmt.Errorf("malformed code while reading LOAD_CONST operand")
				}
				idx := binary.LittleEndian.Uint16(out[i : i+2])
				newIdx := idx + offset
				binary.LittleEndian.PutUint16(out[i:i+2], newIdx)
				i += 2
			case vm.OpStoreLocal, vm.OpLoadLocal, vm.OpJump, vm.OpJumpIfFalse:
				// u16 operan
				if i+2 > len(out) {
					return nil, fmt.Errorf("malformed code while reading u16 operand for op %d", op)
				}
				i += 2
			case vm.OpCallBuiltin:
				// single u8 operand (argc)
				if i+1 > len(out) {
					return nil, fmt.Errorf("malformed code while reading CALL operand")
				}
				i += 1
			case vm.OpAdd, vm.OpSub, vm.OpMul, vm.OpDiv, vm.OpPop:
				// no inline operands
			default:
				return nil, fmt.Errorf("unknown opcode %d while remapping", op)
			}
		}
		return out, nil
	}

	processBlob := func(blob []byte) error {
		chk := Checksum(blob)
		if processed[chk] {
			return nil
		}
		processed[chk] = true

		code, consts, err := vm.DeserializeBytecode(blob)
		if err != nil {
			return fmt.Errorf("deserialize failed: %w", err)
		}

		offset := uint16(len(finishedConsts))

		remappedCode, err := remapLoadConstIndices(code, offset)
		if err != nil {
			return fmt.Errorf("remap error: %w", err)
		}

		for _, c := range consts {
			finishedConsts = append(finishedConsts, c)
		}
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
					Log("warning: gluon had no source.glue: " + path)
				}
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
	finishedCode = append(finishedCode, byte(vm.OpHalt))

	// 4) serialize combined consts + combined code into final blob
	finalBlob, err := vm.SerializeBytecode(finishedCode, finishedConsts)
	if err != nil {
		return fmt.Errorf("serialize failed: %w", err)
	}

	// 5) hand to MakeGluon
	return MakeGluon(finalBlob)
}

