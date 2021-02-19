package tgo

import (
	"bytes"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	Letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!#$%&*+=?@^_|-"
)

func UtilIsEmpty(data string) bool {
	return strings.Trim(data, " ") == ""
}

func UtilGetStringFromIntArray(data []int, sep string) string {

	dataStr := UtilGetStringArrayFromIntArray(data)

	return strings.Join(dataStr, sep)

}

func UtilGetStringFromInt64Array(data []int64, sep string) string {

	dataStr := UtilGetStringArrayFromInt64Array(data)

	return strings.Join(dataStr, sep)

}

func UtilGetStringArrayFromIntArray(data []int) []string {

	model := []string{}

	for _, item := range data {

		m := strconv.Itoa(item)

		model = append(model, m)

	}
	return model
}
func UtilGetStringArrayFromInt64Array(data []int64) []string {

	model := []string{}

	for _, item := range data {

		m := strconv.FormatInt(item, 10)

		model = append(model, m)

	}
	return model
}

func UtilSplitToIntArray(data string, sep string) []int {
	var model []int

	dataArray := strings.Split(data, sep)

	for _, item := range dataArray {
		m, err := strconv.Atoi(item)

		if err != nil {
			continue
		}

		model = append(model, m)
	}
	return model
}

func UtilSplitToInt64Array(data string, sep string) []int64 {
	var model []int64

	dataArray := strings.Split(data, sep)

	for _, item := range dataArray {
		m, err := strconv.ParseInt(item, 10, 64)

		if err != nil {
			continue
		}

		model = append(model, m)
	}
	return model
}

func UtilStringGenerateRandomString(n int) string {
	letters := []rune(Letters)
	rand.Seed(time.Now().UTC().UnixNano())
	randomString := make([]rune, n)
	for i := range randomString {
		randomString[i] = letters[rand.Intn(len(letters))]
	}
	return string(randomString)
}

func UtilStringCheckStringExisted(strs []string, str string) bool {
	for _, v := range strs {
		if v == str {
			return true
		}
	}

	return false
}

func UtilStringContains(obj interface{}, target interface{}) bool {
	targetValue := reflect.ValueOf(target)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < targetValue.Len(); i++ {
			if targetValue.Index(i).Interface() == obj {
				return true
			}
		}
	case reflect.Map:
		if targetValue.MapIndex(reflect.ValueOf(obj)).IsValid() {
			return true
		}
	}
	return false
}

func UtilStringConcat(buffer *bytes.Buffer, str string) {
	buffer.WriteString(str)
}

func UtilStringConcatExist(strs []string, str string) []string {
	return append(strs, str)
}
