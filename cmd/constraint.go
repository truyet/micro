package cmd

import (
	"errors"
	"os"

	ccli "github.com/micro/cli"
)

// var defination
var (
	args        []string //original parameters from os.Args
	mainFlags   []string //mainFlags parsed from os.Args
	subCmd      []string //subCmd parsed from os.Args
	subCmdFlags []string //subCmdFlags for subCmd parsed from os.Args

	ErrHelp                          = errors.New("flag: help requested")
	ErrParsedOver                    = errors.New("arguments: parsed over")
	ErrParsedDoubleStrike            = errors.New("warning :unexpacted flag --")
	ErrParsedNoMainFlagValue         = errors.New("warning :no value found in Mian flag")
	ErrParsedNoSubCmdFlagValue       = errors.New("warning :no value found in SubCmd flag")
	errorLastMainFlag          error = nil
	errorLastSubCmdFlag        error = nil
)

func regularArguments(app *ccli.App) {

	//point to original parameters
	args = os.Args[1:]

	for _, item := range args {
		seen, err := parseOne(item, app.Flags)
		if seen {
			args = args[1:]
			continue
		}
		if !seen && err == nil {
			break
		}
	}

	var newArgs []string
	newArgs = append(newArgs, os.Args[0])
	newArgs = append(newArgs, mainFlags...)
	newArgs = append(newArgs, subCmd...)
	newArgs = append(newArgs, subCmdFlags...)

	os.Args = newArgs
}

// parseOne parses one flag. It reports whether a flag was seen.
func parseOne(s string, appFlags []ccli.Flag) (bool, error) {
	if len(args) == 0 {
		return false, ErrParsedOver
	}

	//check last errorflags is novalue
	if s[0] != '-' {
		if errors.Is(errorLastMainFlag, ErrParsedNoMainFlagValue) {
			mainFlags = append(mainFlags, s)
			errorLastMainFlag = nil
			return true, nil
		}
		if errors.Is(errorLastSubCmdFlag, ErrParsedNoSubCmdFlagValue) {
			subCmdFlags = append(subCmdFlags, s)
			errorLastSubCmdFlag = nil
			return true, nil
		}

	}
	errorLastMainFlag = nil
	errorLastSubCmdFlag = nil

	// merge subcmd
	if len(s) < 2 || s[0] != '-' {
		subCmd = append(subCmd, s)
		return true, nil
	}
	numMinuses := 1
	if s[1] == '-' {
		numMinuses++
		if len(s) == 2 { // "--" terminates the flags
			//	args = args[1:]
			return true, ErrParsedDoubleStrike
		}
	}
	name := s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return true, errors.New("warning :unexpacted bad flag syntax: " + s)
	}

	hasValue := false
	//value := ""
	for i := 1; i < len(name); i++ { // equals cannot be first
		if name[i] == '=' {
			//		value = name[i+1:]
			hasValue = true // equals found
			name = name[0:i]
			break
		}
	}

	//	alreadythere := false
	for _, f := range appFlags {
		if name == f.GetName() {
			//		alreadythere = true              //belong to mainflag
			mainFlags = append(mainFlags, s) //add to mainflag set
			//check the next value
			if !hasValue {
				errorLastMainFlag = ErrParsedNoMainFlagValue
				return true, ErrParsedNoMainFlagValue
			}
			return true, nil
		}
	}

	// clearly we did not find s in appFlags which must be subcmdflag
	//check the next value
	subCmdFlags = append(subCmdFlags, s) //add to flag set
	if !hasValue {
		errorLastMainFlag = ErrParsedNoSubCmdFlagValue
		return true, ErrParsedNoSubCmdFlagValue
	}
	//find a value
	return true, nil

}
