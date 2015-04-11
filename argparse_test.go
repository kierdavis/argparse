package argparse

import (
	"errors"
	"testing"
)

func no_exit(exit_status int) {
	panic(exit_status)
}

func TestParse(t *testing.T) {
	exitFunc = no_exit
	var parser *ArgumentParser = New("pouet")
	parser.Option('b', "by", "By", 1, Store, "", "By")
	var pargs struct {
		By string
	}

	parse := func(v interface{}, s []string) (err error) {
		defer func() {
			if x := recover(); nil != x {
				err = errors.New("panic caught")
			}
		}()
		return parser.ParseArgs(v, s)
	}
	testokfunc := func(a []string) {
		err := parse(&pargs, a)
		if nil != err {
			t.Error("Can not parse cmdline", a, err)
		}
	}
	testnokfunc := func(a []string) {
		err := parse(&pargs, a)
		if nil == err {
			t.Error("Can not parse cmdline", a, err)
		}
	}

	testokfunc([]string{"-b", "pouet"})
	testokfunc([]string{})
	testokfunc([]string{"--", "--truc"})
	testokfunc([]string{"--", "-t"})
	testnokfunc([]string{"-t"})
	testnokfunc([]string{"--truc"})
}

func TestAppendConst(t *testing.T) {
	exitFunc = no_exit
	var parser *ArgumentParser = New("appendconsttest")

	parser.Option('a', "aa", "Params", 0, AppendConst("aflag"), "", "Aflag")
	parser.Option('b', "bb", "Params", 0, AppendConst("bflag"), "", "Bflag")
	parser.Option('c', "cc", "Params", 0, AppendConst("cflag"), "", "Cflag")

	var pargs struct {
		 Params []string
	}

	parse := func(v interface{}, s []string) (err error) {
		defer func() {
			if x := recover(); nil != x {
				err = errors.New("panic caught")
			}
		}()
		return parser.ParseArgs(v, s)
	}
	testokfunc := func(a []string) {
		err := parse(&pargs, a)
		if nil != err {
			t.Error("Can not parse cmdline", a, err)
		}
	}

	testokfunc([]string{"-a", "-a", "-a"})
	if 3 != len(pargs.Params) {
		t.Error("Wrong number of arguments: ", pargs.Params)
	}
	for i, val := range pargs.Params {
		if "aflag" != val {
			t.Error("Wrong argument ", i, "value: ", val)
		}
	}
	pargs.Params = []string{}

	testokfunc([]string{"-a", "-b", "-c", "-c", "-b", "-a"})
	expected := []string{"aflag", "bflag", "cflag", "cflag", "bflag", "aflag"}
	if len(expected) != len(pargs.Params) {
		t.Error("Wrong number of arguments: ", pargs.Params)
	}
	for i, val := range pargs.Params {
		if val != expected[i] {
			t.Error("Wrong argument ", i, "value: ", val)
		}
	}
	pargs.Params = []string{}
}
