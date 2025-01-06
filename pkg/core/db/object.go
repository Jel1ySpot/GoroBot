package db

import (
	"fmt"
	"time"
)

// ObjectRow 行数据
type ObjectRow map[string]Object

// ObjectType 对象类型
type ObjectType int

const (
	Integer ObjectType = iota
	Float
	String
	Time
	List
)

type Object struct {
	Type ObjectType
	Data any
}

func (obj Object) String() string {
	if obj.Data == nil {
		return "null"
	}
	switch obj.Type {
	case Integer:
		return fmt.Sprintf("%d", obj.Data.(int64))
	case Float:
		return fmt.Sprintf("%f", obj.Data.(float64))
	case String:
		return fmt.Sprintf("%s", obj.Data.(string))
	case Time:
		return fmt.Sprintf("%s", obj.Data.(time.Time).Format(time.RFC3339))
	case List:
		return fmt.Sprintf("%v", obj.Data.([]string))
	default:
		return fmt.Sprintf("%v", obj.Data)
	}
}

func (obj Object) Get() any {
	return obj.Data
}

func (obj Object) GetInteger() (int64, bool) {
	val, ok := obj.Data.(int64)
	return val, ok
}

func (obj Object) GetFloat() (float64, bool) {
	val, ok := obj.Data.(float64)
	return val, ok
}

func (obj Object) GetString() (string, bool) {
	val, ok := obj.Data.(string)
	return val, ok
}

func (obj Object) GetTime() (time.Time, bool) {
	val, ok := obj.Data.(time.Time)
	return val, ok
}

func (obj Object) GetList() ([]string, bool) {
	val, ok := obj.Data.([]string)
	return val, ok
}
