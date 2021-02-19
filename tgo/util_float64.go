package tgo

import (
	"fmt"
	"strconv"
)

func UtilFloat64ToInt(value float64, multiplied float64) (intValue int, err error) {

	aString := fmt.Sprintf("%.0f", value*multiplied)

	intValue, err = strconv.Atoi(aString)

	if err != nil {
		LogErrorw(LogNameLogic, "UtilFloat64ToInt error", err)
	}
	return
}
