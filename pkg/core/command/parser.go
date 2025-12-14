package command

import (
	"regexp"
	"strconv"
)

var inputTypes map[InputType]*regexp.Regexp

const (
	Number  InputType = "number"
	String  InputType = "string"
	Boolean InputType = "bool"
)

func GetInt(value string) int {
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return result
}

func GetFloat(value string) float64 {
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return result
}

func GetBool(value string) bool {
	switch value {
	case "T", "TRUE", "True", "true", "t", "on", "ON", "Y", "y", "Yes", "yes", "YES":
		return true
	case "F", "FALSE", "False", "false", "f", "off", "OFF", "N", "n", "No", "no", "NO":
		return false
	}
	return false
}

func CheckInputType(value string, inputType InputType) bool {
	if inputTypes[inputType] != nil {
		return inputTypes[inputType].MatchString(value)
	}
	return false
}

func RegisterInputType(typeName InputType, checker *regexp.Regexp) {
	inputTypes[typeName] = checker
}

func init() {
	inputTypes = make(map[InputType]*regexp.Regexp)

	RegisterInputType(Number, regexp.MustCompile("^[0-9]+$"))
	RegisterInputType(String, regexp.MustCompile("^.+$"))
	RegisterInputType(Boolean, regexp.MustCompile("^(?:[TFtfYNyn]|on|ON|off|OFF|Yes|YES|No|NO|(?:T|t)rue|TRUE|(?:F|f)alse|FALSE)$"))
}
