package qbot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tencent-connect/botgo/constant"
	"github.com/tencent-connect/botgo/openapi"
	"net/http"
	"reflect"
	"strings"
	"unsafe"
)

func NativePost(api openapi.OpenAPI, endpoint string, body interface{}, result interface{}, params map[string]string) (interface{}, error) {
	if params == nil {
		params = make(map[string]string)
	}

	url := getURL(api, endpoint)
	for k, v := range params {
		url = strings.Replace(url, "{"+k+"}", v, -1)
	}
	resp, err := api.Transport(context.Background(), http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(resp, result); err != nil {
		return nil, err
	}
	return result, nil
}

func GetValFromFramework(api openapi.OpenAPI, key string) unsafe.Pointer {
	v := reflect.ValueOf(api).Elem()

	field := v.FieldByName(key)

	if field.IsValid() {
		// 读取未导出字段的值
		return unsafe.Pointer(field.UnsafeAddr())
	}
	return nil
}

func getURL(api openapi.OpenAPI, endpoint string) string {
	d := constant.APIDomain
	if val := GetValFromFramework(api, "url"); val != nil && *(*bool)(val) {
		d = constant.SandBoxAPIDomain
	}
	return fmt.Sprintf("%s%s", d, endpoint)
}
