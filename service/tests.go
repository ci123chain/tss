package service

import "go-api-frame/pconst"

func Test(param string) (int, string) {
	return pconst.CODE_COMMON_OK, param
}
