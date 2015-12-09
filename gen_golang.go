package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"strings"
	"text/template"
)

var go_template = `
package {{.Package}}

import "github.com/funny/binary"

{{range .Structs}}
func (s *{{.Name}}) BinarySize() (n int) {
` + fuck(`
	n = 0
	{{range .Fields}}
		{{if and .Size (not .IsArray)}}
			+ {{.Size}}
		{{else if and .Size .ArraySize}}
			+ {{.Size}} * {{.ArraySize}}
		{{end}}
	{{end}}
	{{range .Fields}}
		{{if and .Size .IsArray (not .ArraySize)}}
			+ {{.Size}} * len(s.{{.Name}})
		{{else if not .IsArray}} 
			{{if or (eq .Type "string") (eq .Type "[]byte")}}
				+ len(s.{{.Name}})
			{{else if .IsUnknow}}
				+ s.{{.Name}}.BinarySize()
			{{end}}
		{{end}}
	{{end}}
`) + `
	{{range .Fields}}
		{{if .IsArray}}
			{{if or (eq .Type "string") (eq .Type "[]byte")}}
				for i := 0; i < {{.GoLen}}; i ++ {
					n += len(s.{{.Name}}[i])
				}
			{{else if .IsUnknow}}
				for i := 0; i < {{.GoLen}}; i ++ {
					n += s.{{.Name}}[i].BinarySize()
				}
			{{end}}
		{{end}}
	{{end}}
	return
}

func (s *{{.Name}}) MarshalBuffer(buf *binary.Buffer) {
	{{range .Fields}}
		{{if .IsArray}}
			{{if not .ArraySize}}
			buf.WriteUint16LE(uint16(len(s.{{.Name}})))
			{{end}}
			for i := 0; i < {{.GoLen}}; i ++ {
				{{.GoEncodeFunc}}
			}
		{{else}}
			{{.GoEncodeFunc}}
		{{end}}
	{{end}}
}

func (s *{{.Name}}) UnmarshalBuffer(buf *binary.Buffer) {
	{{if .GoNeedN}}n := 0{{end}}
	{{range .Fields}}
		{{if .IsArray}}
			{{if not .ArraySize}}
			n = int(buf.ReadUint16LE())
			{{end}}
			for i := 0; i < {{if .ArraySize}}{{.ArraySize}}{{else}}n{{end}}; i ++ {
				{{.GoDecodeFunc}}
			}
		{{else}}
			{{.GoDecodeFunc}}
		{{end}}
	{{end}}
}
{{end}}
`

func fuck(s string) string {
	return strings.Replace(
		strings.Replace(s, "\n", "", -1), "\t", "", -1,
	)
}

func generateGolang(file *File) {
	var bf bytes.Buffer

	tpl := template.Must(template.New("code").Parse(go_template))
	if err := tpl.Execute(&bf, file); err != nil {
		log.Fatalf("Generate code failed: %s", err)
	}

	code, err := format.Source(bf.Bytes())
	if err != nil {
		fmt.Print(bf.String())
		log.Fatalf("Could't format source: %s", err)
	}

	code = bytes.Replace(code, []byte("\n\n"), []byte("\n"), -1)

	if len(flag.Args()) == 0 {
		filename := strings.Replace(file.Name, ".go", ".fast.go", 1)
		file, err := os.Create(filename)
		if err != nil {
			log.Fatalf("Could't create file '%s': %s", filename, err)
		}
		if _, err := file.Write(code); err != nil {
			log.Fatalf("Write file '%s' failed: %s", filename, err)
		}
		file.Close()
	} else {
		fmt.Print(string(code))
	}
}

func (field *Field) GoLen() string {
	if field.ArraySize == "" {
		return fmt.Sprintf("len(s.%s)", field.Name)
	} else {
		return field.ArraySize
	}
}

func (s *Struct) GoNeedN() bool {
	for _, field := range s.Fields {
		if field.IsArray {
			if field.Size == "" || field.ArraySize == "" {
				return true
			}
		}
	}
	return false
}

func (field *Field) GoEncodeFunc() string {
	f := field.Name
	if field.IsArray {
		f += "[i]"
	}
	switch field.Type {
	case "int":
		return fmt.Sprintf("buf.WriteIntLE(s.%s)", f)
	case "uint":
		return fmt.Sprintf("buf.WriteUintLE(s.%s)", f)
	case "int8":
		return fmt.Sprintf("buf.WriteInt8(s.%s)", f)
	case "uint8", "byte":
		return fmt.Sprintf("buf.WriteUint8(s.%s)", f)
	case "int16":
		return fmt.Sprintf("buf.WriteInt16LE(s.%s)", f)
	case "uint16":
		return fmt.Sprintf("buf.WriteUint16LE(s.%s)", f)
	case "int32":
		return fmt.Sprintf("buf.WriteInt32LE(s.%s)", f)
	case "uint32":
		return fmt.Sprintf("buf.WriteUint32LE(s.%s)", f)
	case "int64":
		return fmt.Sprintf("buf.WriteInt64LE(s.%s)", f)
	case "uint64":
		return fmt.Sprintf("buf.WriteUint64LE(s.%s)", f)
	case "string":
		return fmt.Sprintf("buf.WriteUint16LE(uint16(len(s.%s)))\nbuf.WriteString(s.%s)", f, f)
	case "[]byte":
		if field.ArraySize == "" {
			return fmt.Sprintf("buf.WriteUint16LE(uint16(len(s.%s)))\nbuf.WriteBytes(s.%s)", f, f)
		} else {
			return fmt.Sprintf("buf.WriteBytes(s.%s[:])", f)
		}
	}
	return fmt.Sprintf("s.%s.MarshalBuffer(buf)", f)
}

func (field *Field) GoDecodeFunc() string {
	f := field.Name
	if field.IsArray {
		f += "[i]"
	}
	switch field.Type {
	case "int":
		return fmt.Sprintf("s.%s = buf.ReadIntLE()", f)
	case "uint":
		return fmt.Sprintf("s.%s = buf.ReadUintLE()", f)
	case "int8":
		return fmt.Sprintf("s.%s = buf.ReadInt8()", f)
	case "uint8", "byte":
		return fmt.Sprintf("s.%s = buf.ReadUint8()", f)
	case "int16":
		return fmt.Sprintf("s.%s = buf.ReadInt16LE()", f)
	case "uint16":
		return fmt.Sprintf("s.%s = buf.ReadUint16LE()", f)
	case "int32":
		return fmt.Sprintf("s.%s = buf.ReadInt32LE()", f)
	case "uint32":
		return fmt.Sprintf("s.%s = buf.ReadUint32LE()", f)
	case "int64":
		return fmt.Sprintf("s.%s = buf.ReadInt64LE()", f)
	case "uint64":
		return fmt.Sprintf("s.%s = buf.ReadUint64LE()", f)
	case "string":
		return fmt.Sprintf("s.%s = buf.ReadString(int(buf.ReadUint16LE()))", f)
	case "[]byte":
		if field.ArraySize == "" {
			return fmt.Sprintf("s.%s = buf.ReadBytes(int(buf.ReadUint16LE()))", f)
		} else {
			return fmt.Sprintf("copy(s.%s[:], buf.Take(%s))", f, field.ArraySize)
		}
	}
	return fmt.Sprintf("s.%s.UnmarshalBuffer(buf)", f)
}
