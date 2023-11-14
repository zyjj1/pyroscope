// based on https://github.com/grafana/beyla/blob/6b46732da73f2f2cb84e41efdc74789509a7fa2b/pkg/internal/goexec/structmembers.go
package main

import (
	"debug/dwarf"
	"debug/elf"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Field struct {
	Name   string
	Offset uint64

	//attrs map[dwarf.Attr]any
}
type Typedef struct {
	Name        string
	TypeOffsets []dwarf.Offset
}
type Typ struct {
	Name   string
	Fields []Field
	Size   int64
}

func (t Typ) GetField(name string) *Field {
	for i, field := range t.Fields {
		if field.Name == name {
			return &t.Fields[i]
		}
	}
	return nil
}

type Index struct {
	offset2Type map[dwarf.Offset]*Typ
	typedefs    map[string]*Typedef
}

func (i *Index) GetTypeByName2(name string) *Typ {
	if name == "" {
		return nil
	}
	typedef := i.typedefs[name]
	if typedef == nil {
		return i.GetTypeByName(name)
	}
	var res []*Typ
	for _, offset := range typedef.TypeOffsets {
		typ := i.offset2Type[offset]
		if typ == nil {
			panic(fmt.Sprintf("%s %d not found", name, offset))
		}
		res = append(res, typ)
		if len(res) > 0 {
			prev := res[len(res)-1]
			if !reflect.DeepEqual(prev, typ) {
				panic(fmt.Sprintf("not eq %v prev %v", typ, prev))
			}
		}
	}
	if len(res) == 0 {
		return nil
	}
	return res[0]
}
func (i *Index) GetTypeByName(name string) *Typ {
	var res []*Typ
	for _, typ := range i.offset2Type {
		if typ.Name == name {
			res = append(res, typ)
			if len(res) > 0 {
				prev := res[len(res)-1]
				if !reflect.DeepEqual(prev, typ) {
					panic(fmt.Sprintf("not eq %v prev %v", typ, prev))
				}
			}
		}
	}
	if len(res) == 0 {
		return nil
		//panic(fmt.Sprintf("%s not found", name))
	}
	return res[0]
}

// structMemberOffsetsFromDwarf reads the executable dwarf information to get
// the offsets specified in the structMembers map
func structMemberOffsetsFromDwarf(data *dwarf.Data) (Index, error) {

	reader := data.Reader()
	res := Index{
		//name2Type : map[string]*Typ{},
		offset2Type: map[dwarf.Offset]*Typ{},
		typedefs:    map[string]*Typedef{},
	}

	for {
		entry, err := reader.Next()
		if err != nil {
			return res, err
		}
		if entry == nil { // END of dwarf data
			return res, nil
		}
		attrs := getAttrs(entry)

		if entry.Tag != dwarf.TagStructType && entry.Tag != dwarf.TagTypedef {
			continue
		}
		if entry.Tag == dwarf.TagTypedef {
			typeName, _ := attrs[dwarf.AttrName].(string)
			if typeName != "" {
				typedef := res.typedefs[typeName]
				if typedef == nil {
					typedef = &Typedef{Name: typeName}
					res.typedefs[typeName] = typedef
				}
				tt := attrs[dwarf.AttrType]
				if tt != nil {
					typedef.TypeOffsets = append(typedef.TypeOffsets, tt.(dwarf.Offset))
				} else {
					//fmt.Println("hek")
				}
			}
			continue
		}
		typeName, _ := attrs[dwarf.AttrName].(string)

		sz, _ := attrs[dwarf.AttrByteSize].(int64)
		if sz == 0 {
			reader.SkipChildren()
			continue
		}

		offsets, err := readMembers(reader)
		if err != nil {
			return res, err
		}
		nt := &Typ{
			Name:   typeName,
			Size:   sz,
			Fields: offsets,
		}
		res.offset2Type[(entry.Offset)] = nt

	}
}

func readMembers(
	reader *dwarf.Reader,
) ([]Field, error) {
	var res []Field
	for {
		entry, err := reader.Next()
		if err != nil {
			return res, fmt.Errorf("can't read DWARF data: %w", err)
		}
		if entry == nil { // END of dwarf data
			return res, nil
		}
		// Nil tag: end of the members list
		if entry.Tag == 0 {
			return res, nil
		}
		attrs := getAttrs(entry)
		name, nok := attrs[dwarf.AttrName].(string)
		value, vok := attrs[dwarf.AttrDataMemberLoc]
		//fmt.Printf("    %s %d\n", name, value)
		if nok && vok {

			res = append(res, Field{
				name,
				uint64(value.(int64)),
			})
		}
	}
}
func getAttrs(entry *dwarf.Entry) map[dwarf.Attr]any {
	attrs := map[dwarf.Attr]any{}
	for f := range entry.Field {
		attrs[entry.Field[f].Attr] = entry.Field[f].Val
	}
	return attrs
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide python / libpython elf files.")
		return
	}
	var es []Entry
	for _, fp := range os.Args[1:] {
		es = append(es, printOne(fp))
	}
	sort.Slice(es, func(i, j int) bool {
		return es[i].PythonVersion.Compare(es[j].PythonVersion) > 0
	})
	const header = `// Code generated by python_dwarfdump. DO NOT EDIT.
package python

var pyVersions = map[Version]*UserOffsets{`
	fmt.Println(header)
	for _, e := range es {
		fmt.Println(e.Offsets)
	}
	fmt.Println("}")

}

var interesetd = []Need{

	{Name: "PyVarObject", Fields: []NeedField{
		{"ob_size", "PyVarObject_ob_size"},
	}},
	{Name: "PyObject", Fields: []NeedField{
		{"ob_type", "PyObject_ob_type"},
	}},
	{Name: "_typeobject", PrettyName: "PyTypeObject", Fields: []NeedField{
		{"tp_name", "PyTypeObject_tp_name"},
	}},
	{Name: "PyThreadState", Fields: []NeedField{
		{"frame", "PyThreadState_frame"},
		{"cframe", "PyThreadState_cframe"},
		{"current_frame", "PyThreadState_current_frame"},
	}},
	{Name: "_PyCFrame", Fields: []NeedField{
		{"current_frame", "PyCFrame_current_frame"},
	}},
	//typedef struct _frame PyFrameObject;
	{Name: "_frame", PrettyName: "PyFrameObject", Fields: []NeedField{
		{"f_back", "PyFrameObject_f_back"},
		{"f_code", "PyFrameObject_f_code"},
		{"f_localsplus", "PyFrameObject_f_localsplus"},
	}},
	{Name: "PyCodeObject", Fields: []NeedField{
		{"co_filename", "PyCodeObject_co_filename"},
		{"co_name", "PyCodeObject_co_name"},
		{"co_varnames", "PyCodeObject_co_varnames"},
		{"co_localsplusnames", "PyCodeObject_co_localsplusnames"},
	}},
	{Name: "PyTupleObject", Fields: []NeedField{
		{"ob_item", "PyTupleObject_ob_item"},
	}},
	{Name: "_PyInterpreterFrame", Fields: []NeedField{
		{"f_code", "PyInterpreterFrame_f_code"},
		{"f_executable", "PyInterpreterFrame_f_executable"},
		{"previous", "PyInterpreterFrame_previous"},
		{"localsplus", "PyInterpreterFrame_localsplus"},
		{"owner", "PyInterpreterFrame_owner"},
	}},
	{Name: "_PyRuntimeState", Fields: []NeedField{
		{"gilstate", "PyRuntimeState_gilstate"},
		{"autoTSSkey", "PyRuntimeState_autoTSSkey"},
	}},
	{Name: "_gilstate_runtime_state", Fields: []NeedField{
		{"autoTSSkey", "Gilstate_runtime_state_autoTSSkey"},
	}},
	{Name: "_Py_tss_t", Size: true, Fields: []NeedField{
		{"_is_initialized", "PyTssT_is_initialized"},
		{"_key", "PyTssT_key"},
	}},
	{Name: "PyASCIIObject", PrettyName: "PyASCIIObject", Size: true},
	{Name: "PyCompactUnicodeObject", PrettyName: "PyCompactUnicodeObject", Size: true},

	//{Name: "_is", PrettyName: "PyInterpreterState", Fields: []string{}},
}

type PythonVersion struct {
	Major, Minor, Patch int
}

type Entry struct {
	PythonVersion PythonVersion
	Offsets       string
}

func (p *PythonVersion) Compare(other PythonVersion) int {
	major := other.Major - p.Major
	if major != 0 {
		return major
	}

	minor := other.Minor - p.Minor
	if minor != 0 {
		return minor
	}
	return other.Patch - p.Patch
}

func printOne(fp string) Entry {
	//fmt.Println(fp)
	re := regexp.MustCompile("(\\d+)\\.(\\d+)\\.(\\d+)")
	version := re.FindAllSubmatch([]byte(fp), -1)
	if len(version) != 1 {
		panic("no version found" + fp)
	}

	iversion := PythonVersion{}
	var err error
	iversion.Major, err = strconv.Atoi(string(version[0][1]))
	iversion.Minor, err = strconv.Atoi(string(version[0][2]))
	iversion.Patch, err = strconv.Atoi(string(version[0][3]))

	f, err := elf.Open(fp)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	d, err := f.DWARF()
	if err != nil {
		panic(err)

	}

	types, err := structMemberOffsetsFromDwarf(d)
	if err != nil {
		panic(err)
	}

	res := ""
	res += fmt.Sprintf("// %s %s\n", string(version[0][0]), fp)
	res += fmt.Sprintf("{%s, %s, %s}:  {\n", version[0][1], version[0][2], version[0][3])
	for _, need := range interesetd {
		typ := types.GetTypeByName2(need.Name)
		if typ == nil {
			typ = types.GetTypeByName2(need.PrettyName)
		}
		//if typ == nil {
		//	panic(fmt.Sprintf("typ %s not found", need.Name))
		//}

		for _, needField := range need.Fields {
			o := -1
			if typ != nil {
				f := typ.GetField(needField.Name)
				if f != nil {
					o = int(f.Offset)
				}
			}
			pname := needField.PrintName
			if pname == "" {
				pname = fmt.Sprintf("%s%s", typeName(need), fieldName(needField.Name))
			}
			res += fmt.Sprintf("  %s:%d,\n", pname, o)
		}
		if need.Size {
			if typ == nil {
				res += fmt.Sprintf("  %sSize:%d,\n", typeName(need), -1)
			} else {
				res += fmt.Sprintf("  %sSize:%d,\n", typeName(need), typ.Size)
			}
		}
	}
	res += fmt.Sprintf("},")
	return Entry{iversion, res}

}

func typeName(need Need) string {
	n := need.Name
	if need.PrettyName != "" {
		n = need.PrettyName
	}
	n = strings.TrimPrefix(n, "_")
	parts := strings.Split(n, "_")
	for i := range parts {
		p1 := parts[i][:1]
		p2 := parts[i][1:]
		parts[i] = strings.ToUpper(p1) + p2
	}
	return strings.Join(parts, "")

}

func fieldName(field string) string {
	field = strings.TrimPrefix(field, "_")
	parts := strings.Split(field, "_")
	for i := range parts {
		p1 := parts[i][:1]
		p2 := parts[i][1:]
		parts[i] = strings.ToUpper(p1) + p2
	}
	return strings.Join(parts, "")
}

type Need struct {
	Name       string
	PrettyName string
	Fields     []NeedField
	Size       bool
}
type NeedField struct {
	Name      string
	PrintName string
}