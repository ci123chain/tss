package tgo

//验证签名

import (
	"github.com/gin-gonic/gin"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	signAppSecretKey string
	signSwitch       string
	appLimitTime     int
	err              error
)

const (
	SIGN_VERSION_OLD = 1
	SIGN_VERSION_NEW = 2

	SIGN_VERSION_NEW_PREFIX = "1_"
)

/*
sha1的签名算法
   appsecret = ""
   老版本 signature = sha1(appsecret+"babybirthday=1457578839&city=南京市&mobile=15324893018&province=江苏省
&signtimestamp=1457578839&username=张三"+appsecret)
   新版本 signature = '1_' + sha1(appsecret+"babybirthday=1457578839&city=南京市&guid=2793SDG87SDFHG888&mobile=15324893018&province=江苏省
&signtimestamp=1457578839&username=张三"+appsecret)

1. 签名: signature, 时间戳: timestamp, guid：一定存在，优先取query，不存在则从cookie取
2. 参数列表按参数Key字典序升序排列
3. 编码使用 UTF-8
*/

func UtilSignCheckSign(c *gin.Context, token string) bool {
	if signSwitch == "0" {
		return true
	}
	if token == "" {
		token = signAppSecretKey
	}
	signCookie, err := c.Cookie("signature")
	if err != nil {
		return false
	}
	//检测新老版本
	signVer := getSignVersion(signCookie)
	switch signVer {
	case SIGN_VERSION_OLD:
		return checkSign(c, signCookie, SIGN_VERSION_OLD, token)
	case SIGN_VERSION_NEW:
		return checkSign(c, signCookie, SIGN_VERSION_NEW, token)
	default:
		return checkSign(c, signCookie, SIGN_VERSION_OLD, token)
	}
}

func checkSign(c *gin.Context, signCookie string, ver int, token string) bool {
	ps := UtilRequestGetAllParams(c)
	signTimestamp, b := UtilSignCheckSignTimestamp(c)
	if !b {
		return false
	}
	ps.Set("signtimestamp", signTimestamp)
	if ver == SIGN_VERSION_NEW && ps.Get("guid") == "" {
		guid, _ := c.Cookie("guid")
		ps.Set("guid", guid)
	}
	sortedParams := UtilSignGetSortUpParamsString(ps)
	signString := token + sortedParams + token
	signature := UtilCryptoSha1(signString)
	if ver == SIGN_VERSION_NEW {
		signature = SIGN_VERSION_NEW_PREFIX + signature
	}
	if signature == signCookie {
		return true
	}
	return false
}

//升序排序的参数拼接的字符串
func UtilSignGetSortUpParamsString(ps url.Values) string {
	var psKey []string
	for k := range ps {
		psKey = append(psKey, k)
	}
	sort.Strings(psKey)
	var ret []string
	for _, v := range psKey {
		ret = append(ret, v+"="+ps.Get(v))
	}
	return strings.Join(ret, "&")
}

//检测请求时间是否有效
func UtilSignCheckSignTimestamp(c *gin.Context) (ts string, b bool) {
	var err error
	ts, err = c.Cookie("signtimestamp")
	if err != nil {
		return
	}
	signTimestamp, err := strconv.Atoi(ts)
	if err != nil {
		return
	}
	if appLimitTime == 0 {
		b = true
		return
	}
	now := time.Now().Unix()
	if now < int64(appLimitTime+signTimestamp) {
		b = true
		return
	}
	return
}

func getSignVersion(signStr string) int {
	if strings.Contains(signStr, SIGN_VERSION_NEW_PREFIX) {
		return SIGN_VERSION_NEW
	} else {
		return SIGN_VERSION_OLD
	}
}
