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
