package validate

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type (
	compare interface {
		lt(control, target string) bool
		le(control, target string) bool
		ge(control, target string) bool
		gt(control, target string) bool
	}
	numCompare  struct{}
	timeCompare struct{}
)

const (
	timeRegexp = `^[[(]\s*(\d+([mdh]|milli)){1}\s*,\s*(\d+([mdh]|milli)){1}\s*[])]$`
	numRegexp  = `^[[(]\s*([+-]?\d+)|~{1}\s*,\s*([+-]?\d+)|~{1}\s*[])]$`
	selfRegexp = `self\.\w+`
)

var regMap = map[string]*regexp.Regexp{
	timeRegexp: regexp.MustCompile(timeRegexp),
	numRegexp:  regexp.MustCompile(numRegexp),
	selfRegexp: regexp.MustCompile(selfRegexp),
}

var errParseFail = errors.New("parse string fail")

func toInt64(val string) int64 {
	res, _ := strconv.ParseInt(val, 10, 64)
	return res
}
func toUint64(val string) (uint64, error) {
	return strconv.ParseUint(val, 10, 64)
}
func toFloat64(val string) (float64, error) {
	return strconv.ParseFloat(val, 64)
}

var (
	numReg = regexp.MustCompile(`\d+`)
)

// parseTimeDuration 将时间字符串解析成 数字+单位
func parseTimeDuration(val string) (num int64, unit string) {
	numStr := numReg.FindString(val)
	num, err := strconv.ParseInt(numStr, 10, 61)
	if err != nil {
		panic(fmt.Errorf("parseTimeDuration err :%s", err))
	}
	return num, val[len(numStr):]
}

// toTimeDuration 将时间字符串转换成 time.Duration
func toTimeDuration(val string) time.Duration {
	var (
		num, unit = parseTimeDuration(val)
	)

	return time.Duration(num) * func() time.Duration {
		switch unit {
		case "milli":
			return time.Millisecond
		case "m":
			return time.Minute
		case "h":
			return time.Hour
		case "d":
			return time.Hour * 24
		default:
			return time.Duration(1)
		}
	}()
}

func (t timeCompare) lt(control, target string) bool {
	return toTimeDuration(control) < toTimeDuration(target)
}

func (t timeCompare) le(control, target string) bool {
	return toTimeDuration(control) <= toTimeDuration(target)
}

func (t timeCompare) ge(control, target string) bool {

	return toTimeDuration(control) >= toTimeDuration(target)
}

func (t timeCompare) gt(control, target string) bool {

	return toTimeDuration(control) > toTimeDuration(target)
}

func getUintNum(control, target string) (c, t uint64, err error) {
	c, cerr := toUint64(control)
	t, terr := toUint64(target)
	if cerr != nil || terr != nil {
		return 0, 0, errParseFail
	}
	return c, t, err
}
func getFloatNum(control, target string) (c, t float64, err error) {
	c, cerr := toFloat64(control)
	t, terr := toFloat64(target)
	if cerr != nil || terr != nil {
		return 0, 0, errParseFail
	}
	return c, t, err
}
func (n numCompare) lt(control, target string) bool {
	if c, t, err := getUintNum(control, target); err == nil {
		return c < t
	}
	if c, t, err := getFloatNum(control, target); err == nil {
		return c < t
	}

	return toInt64(control) < toInt64(target)
}

func (n numCompare) le(control, target string) bool {
	if c, t, err := getUintNum(control, target); err == nil {
		return c <= t
	}
	if c, t, err := getFloatNum(control, target); err == nil {
		return c <= t
	}
	return toInt64(control) <= toInt64(target)
}

func (n numCompare) ge(control, target string) bool {
	if c, t, err := getUintNum(control, target); err == nil {
		return c >= t
	}
	if c, t, err := getFloatNum(control, target); err == nil {
		return c >= t
	}
	return toInt64(control) >= toInt64(target)
}

func (n numCompare) gt(control, target string) bool {
	if c, t, err := getUintNum(control, target); err == nil {
		return c > t
	}
	if c, t, err := getFloatNum(control, target); err == nil {
		return c > t
	}
	return toInt64(control) > toInt64(target)
}

func checkTagValidate(exp string, val string) {
	//根据规则提取关键信息
	if len(regMap[exp].FindString(val)) == 0 {
		panic(fmt.Errorf("validate tag :%s", val))
	}
}

type checkFunc func(field, val, control string) error

type direction int

const (
	left direction = iota + 1
	right
)

func parsePartition(direction direction, val string, comp compare, control string) (success bool, _ string) {
	var rangeKey, target string
	switch direction {
	case left:
		rangeKey, target = val[0:1], val[1:]
	case right:
		rangeKey, target = val[len(val)-1:], val[:len(val)-1]
	}
	if target == "~" {
		return true, ""
	}
	switch rangeKey {
	case "[":
		return comp.ge(control, target), "too low"
	case "(":
		return comp.gt(control, target), "too low"
	case "]":
		return comp.le(control, target), "too high"
	case ")":
		return comp.lt(control, target), "too high"
	default:
		panic("illegal border")
	}
	return
}

func builder(comp compare) checkFunc {
	return func(field, val, control string) error {
		valid := func(direction direction, singlePartition string) error {
			if succ, hint := parsePartition(direction, singlePartition, comp, control); !succ {
				return fmt.Errorf("validate fail: %s is %s", field, hint)
			}
			return nil
		}
		partition := strings.Split(val, ",")
		leftPar, rightPar := partition[0], partition[1]
		if err := valid(left, leftPar); err != nil {
			return err
		}
		if err := valid(right, rightPar); err != nil {
			return err
		}
		return nil
	}
}

// replaceSelfExp 将 self.{字段名} 替换成具体值
func replaceSelfExp(str string, t reflect.Type, v reflect.Value) string {
	list := regMap[selfRegexp].FindAllString(str, -1)
	if len(list) == 0 {
		return str
	}

	for _, s := range list {
		name := strings.Trim(s, "self.")
		_, ok := t.FieldByName(name)
		if !ok {
			panic(fmt.Errorf("field :%s not exist", name))
		}

		value := v.FieldByName(name)
		if !value.IsValid() {
			panic(fmt.Errorf("not found filed :%s", s))
		}
		if !value.CanInt() && !value.CanUint() {
			panic(fmt.Errorf("filed :%s can't convert to int", s))
		}
		str = strings.Replace(str, s, number2Str(value), 1)
	}
	return str
}

func getTypeStruct(t reflect.Type) reflect.Type {
	if t.Kind() != reflect.Struct && t.Kind() != reflect.Pointer {
		panic("validate target must be struct")
	}
	if t.Kind() == reflect.Struct {
		return t
	}
	return getTypeStruct(t.Elem())
}
func getValueStruct(t reflect.Value) reflect.Value {
	if t.Kind() != reflect.Struct && t.Kind() != reflect.Pointer {
		panic("validate target must be struct")
	}
	if t.Kind() == reflect.Struct {
		return t
	}
	return getValueStruct(t.Elem())
}

// Validate 校验结构体的字段是否合法
/*
- 使用 `valid` tag 进行标记
- 对于数字类型使用范围校验，通过 [] () 代表边界 ，数字和可以带符号 [+、-] , 可以使用 ~ 代表无穷
- 对于时间类型，范围校验与数组相同，需要在数字后面加上单位 [milli,m,h,d] ，不支持符号
- 也可以支持与当前结构体的其他字段进行比较，通过 self.{字段名} 指定 ，注意 int 和 time.Duration 也可以进行比较
*/
func Validate(i interface{}) error {
	iType := getTypeStruct(reflect.TypeOf(i))
	iValue := getValueStruct(reflect.ValueOf(i))

	for i := 0; i < iType.NumField(); i++ {
		field := iType.Field(i)

		valid := replaceSelfExp(field.Tag.Get("valid"), iType, iValue)
		if valid == "" {
			continue
		}
		var doCheck checkFunc
		switch {
		case isNumKind[field.Type.String()]:
			checkTagValidate(numRegexp, valid)
			doCheck = builder(numCompare{})
		case field.Type.String() == "time.Duration":
			checkTagValidate(timeRegexp, valid)
			doCheck = builder(timeCompare{})
		}

		value := iValue.Field(i)
		checkNumber(value, field.Name)

		if err := doCheck(field.Name, valid, number2Str(value)); err != nil {
			return err
		}

	}
	return nil
}

func checkNumber(value reflect.Value, name string) {
	if !value.CanInt() && !value.CanUint() && !value.CanFloat() {
		panic(fmt.Errorf("field :%s can't convert to number", name))
	}
}

func number2Str(value reflect.Value) string {
	if value.CanInt() {
		return strconv.FormatInt(value.Int(), 10)
	}
	if value.CanUint() {
		return strconv.FormatUint(value.Uint(), 10)
	}
	if value.CanFloat() {
		return strconv.FormatFloat(value.Float(), 'e', 10, 64)
	}
	fmt.Println("sd")
	panic("getInt err")
}

var isNumKind = map[string]bool{
	"int":     true,
	"int8":    true,
	"int16":   true,
	"int32":   true,
	"int64":   true,
	"uint":    true,
	"uint8":   true,
	"uint16":  true,
	"uint32":  true,
	"uint64":  true,
	"float64": true,
	"float32": true,
}
