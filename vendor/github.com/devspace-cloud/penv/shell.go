package penv

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type (
	shell struct {
		configFileName string
		commentSigil   string
		quote          func(string) string
		mkSet          func(*shell, NameValue) string
		mkUnset        func(*shell, NameValue) string
		mkAppend       func(*shell, NameValue) string
	}
	shellOp struct {
		op        string
		nameValue NameValue
	}
)

const shellSectionSigil = "#========[ github.com/golang-book/penv ]========="

func (sh *shell) encodeOp(sop shellOp) string {
	return sop.op + ":" +
		hex.EncodeToString([]byte(sop.nameValue.Name)) + ":" +
		hex.EncodeToString([]byte(sop.nameValue.Value))
}

func (sh *shell) decodeOp(ln string) (shellOp, error) {
	var sop shellOp
	i := strings.LastIndex(ln, sh.commentSigil)
	if i < 0 {
		return sop, fmt.Errorf("expected comment")
	}
	args := strings.Split(ln[i+len(sh.commentSigil):], ":")
	if len(args) < 3 {
		return sop, fmt.Errorf("expected 3 arguments")
	}
	sop.op = args[0]
	bs, err := hex.DecodeString(args[1])
	if err != nil {
		return sop, err
	}
	sop.nameValue.Name = string(bs)
	bs, err = hex.DecodeString(args[2])
	if err != nil {
		return sop, err
	}
	sop.nameValue.Value = string(bs)
	return sop, nil
}

func (sh *shell) Load() (*Environment, error) {
	env := &Environment{
		Appenders: make([]NameValue, 0),
		Setters:   make([]NameValue, 0),
		Unsetters: make([]NameValue, 0),
	}

	f, err := os.Open(sh.configFileName)
	if err != nil {
		return env, nil
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	inCode := false
	for s.Scan() {
		if s.Text() == shellSectionSigil {
			inCode = !inCode
		} else if inCode {
			sop, err := sh.decodeOp(s.Text())
			if err != nil {
				continue
			}
			switch sop.op {
			case "SET":
				env.Setters = append(env.Setters, sop.nameValue)
			case "UNSET":
				env.Unsetters = append(env.Unsetters, sop.nameValue)
			case "APPEND":
				env.Appenders = append(env.Appenders, sop.nameValue)
			}
		}
	}

	return env, s.Err()
}

func (sh *shell) Save(env *Environment) error {
	inName := filepath.Join(sh.configFileName)
	outName := "/tmp/penv.tmp"

	// generate the new file
	err := func() error {
		fi, err := os.Stat(inName)
		if err != nil {
			os.MkdirAll(filepath.Dir(inName), 0755)
			ioutil.WriteFile(inName, []byte{}, 0755)
			fi, err = os.Stat(inName)
		}
		if err != nil {
			return err
		}

		in, err := os.Open(inName)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.OpenFile(outName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fi.Mode())
		if err != nil {
			in.Close()
			return err
		}
		defer out.Close()

		w := bufio.NewWriter(out)

		s := bufio.NewScanner(in)
		inCode := false
		for s.Scan() {
			if s.Text() == shellSectionSigil {
				inCode = !inCode
			} else if !inCode {
				_, err = w.WriteString(s.Text() + "\n")
				if err != nil {
					out.Close()
					in.Close()
					os.Remove(outName)
				}
			}
		}

		if s.Err() != nil {
			return s.Err()
		}

		_, err = w.WriteString(shellSectionSigil + "\n")
		if err != nil {
			return err
		}

		for _, nv := range env.Setters {
			_, err = w.WriteString(sh.mkSet(sh, nv) +
				sh.commentSigil + sh.encodeOp(shellOp{
				op:        "SET",
				nameValue: nv,
			}) + "\n")
			if err != nil {
				return err
			}
		}
		for _, nv := range env.Appenders {
			_, err = w.WriteString(sh.mkAppend(sh, nv) +
				sh.commentSigil + sh.encodeOp(shellOp{
				op:        "APPEND",
				nameValue: nv,
			}) + "\n")
			if err != nil {
				return err
			}
		}
		for _, nv := range env.Unsetters {
			_, err = w.WriteString(sh.mkUnset(sh, nv) +
				sh.commentSigil + sh.encodeOp(shellOp{
				op:        "UNSET",
				nameValue: nv,
			}) + "\n")
			if err != nil {
				return err
			}
		}

		_, err = w.WriteString(shellSectionSigil + "\n")
		if err != nil {
			return err
		}

		return w.Flush()
	}()

	if err != nil {
		os.Remove(outName)
		return err
	}

	// if everything is ok overwrite the file
	return os.Rename(outName, inName)
}
