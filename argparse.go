package argparse

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

var nilValue reflect.Value

const (
	Optional   = -1
	OneOrMore  = -2
	ZeroOrMore = -3
)

type CommandLineError string

func (err CommandLineError) Error() (s string) {
	return string(err)
}

type argsList struct {
	items  []string
	ptr    int
	pushed string
}

func (a *argsList) EOF() (eof bool) {
	return a.ptr >= len(a.items)
}

func (a *argsList) Next() (s string) {
	if a.pushed != "" {
		s = a.pushed
		a.pushed = ""
		return s
	}

	if a.EOF() {
		panic(CommandLineError("Not enough arguments"))
	}

	s = a.items[a.ptr]
	a.ptr++
	return s
}

func (a *argsList) Peek() (s string) {
	if a.pushed != "" {
		return a.pushed
	}

	if a.EOF() {
		return ""
	}

	return a.items[a.ptr]
}

func (a *argsList) Push(s string) {
	a.pushed = s
}

func (a *argsList) BackUp() {
	a.ptr--
}

type ArgumentParser struct {
	Description         string
	WordWrapWidth       int
	PositionalArguments []*PositionalArgument
	OptionalArguments   []*OptionalArgument
}

func New(description string) (p *ArgumentParser) {
	p = &ArgumentParser{
		Description:         description,
		WordWrapWidth:       80,
		PositionalArguments: make([]*PositionalArgument, 0),
		OptionalArguments:   make([]*OptionalArgument, 0),
	}

	helpCallback := func(nArgs int, args []string, dest reflect.Value) (err error) {
		p.Help()
		os.Exit(0)
		return nil
	}

	p.Option('h', "help", "", 0, helpCallback, "", "Shows this help message before exiting.")

	return p
}

func (p *ArgumentParser) Error(s string) {
	p.Usage()
	fmt.Printf("\nTry %s --help for help\n\n*** %s\n", os.Args[0], s)
	os.Exit(2)
}

func (p *ArgumentParser) Usage() {
	optionsStr := ""
	if len(p.OptionalArguments) >= 0 {
		optionsStr = " (options)"
	}

	argsStr := ""
	for _, posArg := range p.PositionalArguments {
		argsStr += " " + argsString(posArg.NArgs, posArg.Metavar)
	}

	fmt.Printf("usage: %s%s%s\n", os.Args[0], optionsStr, argsStr)

	if p.Description != "" {
		fmt.Printf("\n%s - %s", os.Args[0], wordWrap(p.Description, p.WordWrapWidth, len(os.Args[0])+3))
	}
}

func (p *ArgumentParser) Help() {
	p.Usage()

	if len(p.PositionalArguments) > 0 {
		fmt.Printf("\nPositional arguments:\n")

		posArgStrs := []string{}
		l := 0

		for _, posArg := range p.PositionalArguments {
			s := fmt.Sprintf("  %s  ", argsString(posArg.NArgs, posArg.Metavar))
			posArgStrs = append(posArgStrs, s)

			if len(s) > l {
				l = len(s)
			}
		}

		for i, posArg := range p.PositionalArguments {
			s := posArgStrs[i]
			fmt.Print(s)
			fmt.Print(strings.Repeat(" ", l-len(s)))
			fmt.Print(wordWrap(posArg.Help, p.WordWrapWidth, l+1))
		}
	}

	if len(p.OptionalArguments) > 0 {
		fmt.Printf("\nOptions:\n")

		optArgStrs := []string{}
		l := 0

		for _, optArg := range p.OptionalArguments {
			var s string

			if optArg.Metavar == "" {
				s = fmt.Sprintf("  -%c, --%s  ", optArg.ShortName, optArg.LongName)
			} else {
				s = fmt.Sprintf("  -%c %s, --%s=%s  ", optArg.ShortName, optArg.Metavar, optArg.LongName, optArg.Metavar)
			}

			optArgStrs = append(optArgStrs, s)

			if len(s) > l {
				l = len(s)
			}
		}

		for i, optArg := range p.OptionalArguments {
			s := optArgStrs[i]
			fmt.Print(s)
			fmt.Print(strings.Repeat(" ", l-len(s)))
			fmt.Print(wordWrap(optArg.Help, p.WordWrapWidth, l+1))
		}
	}
}

func (p *ArgumentParser) Argument(dest string, nArgs int, action ActionFunc, metavar string, help string) {
	p.PositionalArguments = append(p.PositionalArguments, &PositionalArgument{
		Dest:    dest,
		NArgs:   nArgs,
		Action:  action,
		Metavar: metavar,
		Help:    help,
	})
}

func (p *ArgumentParser) Option(shortName byte, longName string, dest string, nArgs int, action ActionFunc, metavar string, help string) {
	p.OptionalArguments = append(p.OptionalArguments, &OptionalArgument{
		ShortName: shortName,
		LongName:  longName,
		Dest:      dest,
		NArgs:     nArgs,
		Action:    action,
		Metavar:   metavar,
		Help:      help,
	})
}

func (p *ArgumentParser) Parse(values interface{}) (err error) {
	/*
		defer func() {
			if x := recover(); x != nil {
				ok := false
				if err, ok = x.(error); ok {
					return
				}
			}
		}()
	*/

	args := &argsList{os.Args[1:], 0, ""}
	dest := reflect.ValueOf(values)
	posArgs := []string{}

	if dest.Type().Kind() == reflect.Ptr {
		dest = dest.Elem()
	}

	for !args.EOF() {
		argStr := args.Next()

		if len(argStr) > 1 && argStr[0] == '-' {
			if argStr[1] == '-' {
				err = p.parseLongOption(argStr[2:], args, dest)
			} else {
				err = p.parseShortOptions(argStr[1:], args, dest)
			}

		} else {
			posArgs = append(posArgs, argStr)
		}

		if err != nil {
			if cmdLineErr, ok := err.(CommandLineError); ok {
				p.Error(string(cmdLineErr))
			}

			return err
		}
	}

	posArgsList := &argsList{posArgs, 0, ""}

	for _, posArg := range p.PositionalArguments {
		err = posArg.parse(posArgsList, dest)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *ArgumentParser) parseShortOptions(s string, args *argsList, dest reflect.Value) (err error) {
	for _, name := range s {
		err = p.parseShortOption(byte(name), args, dest)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *ArgumentParser) parseShortOption(name byte, args *argsList, dest reflect.Value) (err error) {
	for _, arg := range p.OptionalArguments {
		if arg.ShortName == name {
			return arg.parse(args, dest)
		}
	}

	p.Error(fmt.Sprintf("No such option -%c", name))
	return nil
}

func (p *ArgumentParser) parseLongOption(name string, args *argsList, dest reflect.Value) (err error) {
	pos := strings.IndexRune(name, '=')
	if pos >= 0 {
		name = name[:pos]
		args.Push(name[pos+1:])
	}

	for _, arg := range p.OptionalArguments {
		if name == arg.LongName {
			return arg.parse(args, dest)
		}
	}

	p.Error(fmt.Sprintf("No such option --%s", name))
	return nil
}

type PositionalArgument struct {
	NArgs   int
	Dest    string
	Action  ActionFunc
	Metavar string
	Help    string
}

func (arg *PositionalArgument) parse(args *argsList, destStruct reflect.Value) (err error) {
	dest := destStruct.FieldByName(arg.Dest)
	if !dest.IsValid() {
		return fmt.Errorf("Invalid destination struct field: %s", arg.Dest)
	}

	if arg.NArgs == 0 {
		return arg.Action(0, nil, dest)
	}

	return arg.Action(arg.NArgs, readArgStrings(arg.NArgs, args), dest)
}

type OptionalArgument struct {
	ShortName byte
	LongName  string
	NArgs     int
	Dest      string
	Action    ActionFunc
	Metavar   string
	Help      string
}

func (arg *OptionalArgument) parse(args *argsList, destStruct reflect.Value) (err error) {
	dest := destStruct.FieldByName(arg.Dest)
	if !dest.IsValid() {
		return fmt.Errorf("Invalid destination struct field: %s", arg.Dest)
	}

	if arg.NArgs == 0 {
		return arg.Action(0, nil, dest)
	}

	return arg.Action(arg.NArgs, readArgStrings(arg.NArgs, args), dest)
}

type ActionFunc func(int, []string, reflect.Value) error

func Choice(subAction ActionFunc, choices ...string) (action ActionFunc) {
	return func(nArgs int, args []string, value reflect.Value) (err error) {
		for _, arg := range args {
			matched := false

			for _, choice := range choices {
				if choice == arg {
					matched = true
				}
			}

			if !matched {
				return CommandLineError("Expected one of " + strings.Join(choices, ", ") + " for choice argument")
			}
		}

		return subAction(nArgs, args, value)
	}
}

func StoreConst(v interface{}) (action ActionFunc) {
	return func(nArgs int, args []string, value reflect.Value) (err error) {
		value.Set(reflect.ValueOf(v))
		return nil
	}
}

func Store(nArgs int, args []string, value reflect.Value) (err error) {
	t := value.Type()

	if nArgs == 1 || nArgs == Optional {
		if nArgs == Optional && len(args) == 0 {
			return nil // Leave as default
		}

		return storeValue(args[0], value)
	}

	if t.Kind() != reflect.Slice {
		return fmt.Errorf("Invalid kind for Store destination: %s", t.Kind().String())
	}

	slice := reflect.MakeSlice(t, len(args), len(args))

	for i, arg := range args {
		err = storeValue(arg, slice.Index(i))
		if err != nil {
			return err
		}
	}

	value.Set(slice)
	return nil
}

func Append(nArgs int, args []string, value reflect.Value) (err error) {
	t := value.Type()

	if t.Kind() != reflect.Slice {
		return fmt.Errorf("Invalid kind for Append destination: %s", t.Kind().String())
	}

	elemType := t.Elem()

	for _, arg := range args {
		// Create a pointer and immediately dereference, in order to create a writeable value.
		v := reflect.New(elemType).Elem()

		err = storeValue(arg, v)
		if err != nil {
			return err
		}

		value.Set(reflect.Append(value, v))
	}

	return nil
}

func readArgStrings(nArgs int, args *argsList) (argStrings []string) {
	switch nArgs {
	case Optional:
		peek := args.Peek()
		if peek == "" || (len(peek) >= 2 && peek[0] == '-') {
			argStrings = []string{}
		} else {
			argStrings = []string{args.Next()}
		}

	case OneOrMore:
		argStrings = []string{args.Next()}
		peek := args.Peek()

		for !(peek == "" || (len(peek) >= 2 && peek[0] == '-')) {
			argStrings = append(argStrings, args.Next())
			peek = args.Peek()
		}

	case ZeroOrMore:
		peek := args.Peek()

		for !(peek == "" || (len(peek) >= 2 && peek[0] == '-')) {
			argStrings = append(argStrings, args.Next())
			peek = args.Peek()
		}

	default:
		for i := 0; i < nArgs; i++ {
			argStrings = append(argStrings, args.Next())
		}
	}

	return argStrings
}

func storeValue(s string, value reflect.Value) (err error) {
	switch value.Type().Kind() {
	case reflect.Bool:
		var n bool
		n, err = strconv.ParseBool(s)
		value.SetBool(n)

	case reflect.Int:
		var n int64
		n, err = strconv.ParseInt(s, 0, 0)
		value.SetInt(n)

	case reflect.Int8:
		var n int64
		n, err = strconv.ParseInt(s, 0, 8)
		value.SetInt(n)

	case reflect.Int16:
		var n int64
		n, err = strconv.ParseInt(s, 0, 16)
		value.SetInt(n)

	case reflect.Int32:
		var n int64
		n, err = strconv.ParseInt(s, 0, 32)
		value.SetInt(n)

	case reflect.Int64:
		var n int64
		n, err = strconv.ParseInt(s, 0, 64)
		value.SetInt(n)

	case reflect.Uint:
		var n uint64
		n, err = strconv.ParseUint(s, 0, 0)
		value.SetUint(n)

	case reflect.Uint8:
		var n uint64
		n, err = strconv.ParseUint(s, 0, 8)
		value.SetUint(n)

	case reflect.Uint16:
		var n uint64
		n, err = strconv.ParseUint(s, 0, 16)
		value.SetUint(n)

	case reflect.Uint32:
		var n uint64
		n, err = strconv.ParseUint(s, 0, 32)
		value.SetUint(n)

	case reflect.Uint64, reflect.Uintptr:
		var n uint64
		n, err = strconv.ParseUint(s, 0, 64)
		value.SetUint(n)

	case reflect.Float32:
		var n float64
		n, err = strconv.ParseFloat(s, 32)
		value.SetFloat(n)

	case reflect.Float64:
		var n float64
		n, err = strconv.ParseFloat(s, 64)
		value.SetFloat(n)

	case reflect.String:
		value.SetString(s)

	default:
		err = fmt.Errorf("Invalid kind for element destination: %s", value.Type().Kind().String())
	}

	return err
}

func argsString(nArgs int, metavar string) (s string) {
	switch nArgs {
	case Optional:
		return metavar + "?"

	case OneOrMore:
		return metavar + " " + metavar + "..."

	case ZeroOrMore:
		return metavar + "..."
	}

	return metavar
}

func wordWrap(text string, width int, hangingIndentWidth int) (result string) {
	origText := text
	text = strings.Trim(text, " \t\r\n")
	width -= hangingIndentWidth
	if width < 0 { // Can't do much here
		width = 10
	}

	for len(text) > width {
		p := width

		for !unicode.IsSpace(rune(text[p])) {
			p--

			if p < 0 {
				return wordWrap(origText, width+hangingIndentWidth+10, hangingIndentWidth)
			}
		}

		part := strings.TrimRight(text[:p], " \t\r\n")
		//if len(part) == 0 { // Got stuck!
		//return wordWrap(origText, width+hangingIndentWidth+10, hangingIndentWidth)
		//}

		result += part + "\n" + strings.Repeat(" ", hangingIndentWidth)
		text = text[p+1:]
	}

	result += text + "\n"

	return result
}
