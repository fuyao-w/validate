package validate

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type (
	ordered interface {
		lt(control, target interface{}) bool
		le(control, target interface{}) bool
		ge(control, target interface{}) bool
		gt(control, target interface{}) bool
	}
	baseOrdered interface {
		~float64 | ~uint64 | ~int64
	}
	numOrder  struct{}
	direction int
)

const (
	left direction = iota + 1
	right
)

const (
	validTag        = "valid"
	selfPlaceholder = "self."
	// timeRegexp 时间类型正则
	timeRegexp = `^[[(]\s*(\d+([mdh]|milli)){1}\s*,\s*(\d+([mdh]|milli)){1}\s*[])]$`
	// numRegexp 数值类型正则
	numRegexp = `^[[(]\s*([+-]?\d+)|~{1}\s*,\s*([+-]?\d+)|~{1}\s*[])]$`
	// selfRegexp self 类型正则
	selfRegexp = `self\.\w+`
)

var (
	numOrderIns  = new(numOrder)
	durationType = reflect.TypeOf(time.Duration(0))
	numReg       = regexp.MustCompile(`\d+`)
	regMap       = map[string]*regexp.Regexp{
		timeRegexp: regexp.MustCompile(timeRegexp),
		numRegexp:  regexp.MustCompile(numRegexp),
		selfRegexp: regexp.MustCompile(selfRegexp),
	}
	isNumKind = map[reflect.Kind]bool{
		reflect.Int:     true,
		reflect.Int8:    true,
		reflect.Int16:   true,
		reflect.Int32:   true,
		reflect.Int64:   true,
		reflect.Uint:    true,
		reflect.Uint8:   true,
		reflect.Uint16:  true,
		reflect.Uint32:  true,
		reflect.Uint64:  true,
		reflect.Float64: true,
		reflect.Float32: true,
	}
)

func lt[T baseOrdered](a, b T) bool {
	return a < b
}
func le[T baseOrdered](a, b T) bool {
	return a <= b
}
func gt[T baseOrdered](a, b T) bool {
	return a > b
}
func ge[T baseOrdered](a, b T) bool {
	return a >= b
}

func (n *numOrder) lt(control, target interface{}) bool {
	switch t := control.(type) {
	case int64:
		return lt(control.(int64), target.(int64))
	case uint64:
		return lt(control.(uint64), target.(uint64))
	case float64:
		return lt(control.(float64), target.(float64))
	default:
		panic(fmt.Errorf("can't process type :%v", t))
	}
}

func (n *numOrder) le(control, target interface{}) bool {
	switch t := control.(type) {
	case int64, time.Duration:
		return le(control.(int64), target.(int64))
	case uint64:
		return le(control.(uint64), target.(uint64))
	case float64:
		return le(control.(float64), target.(float64))
	default:
		panic(fmt.Errorf("can't process type :%v", t))
	}
}

func (n *numOrder) ge(control, target interface{}) bool {
	switch t := control.(type) {
	case int64:
		return ge(control.(int64), target.(int64))
	case uint64:
		return ge(control.(uint64), target.(uint64))
	case float64:
		return ge(control.(float64), target.(float64))
	default:
		panic(fmt.Errorf("can't process type :%v", t))
	}
}

func (n *numOrder) gt(control, target interface{}) bool {
	switch t := control.(type) {
	case int64:
		return gt(control.(int64), target.(int64))
	case uint64:
		return gt(control.(uint64), target.(uint64))
	case float64:
		return gt(control.(float64), target.(float64))
	default:
		panic(fmt.Errorf("can't process type :%v", t))
	}
}

// parseTimeDuration 将时间字符串解析成 数字+单位
func parseTimeDuration(val string) (num int64, unit string, err error) {
	numStr := numReg.FindString(val)
	num, err = strconv.ParseInt(numStr, 10, 61)
	if err != nil {
		return 0, "", fmt.Errorf("parseTimeDuration err :%s", err)
	}
	return num, val[len(numStr):], nil
}

// checkTagValidate 检查 tag 是否符合语法要求
func checkTagValidate(exp string, val string) {
	//根据规则提取关键信息
	if len(regMap[exp].FindString(val)) == 0 {
		panic(fmt.Errorf("validate tag :%s", val))
	}
}

// parseInt64 将数值或者时间字符串转换成 int64
func parseInt64(val string) (int64, error) {
	var (
		num, unit, err = parseTimeDuration(val)
	)
	return num * func() int64 {
		switch unit {
		case "milli":
			return int64(time.Millisecond)
		case "m":
			return int64(time.Minute)
		case "h":
			return int64(time.Hour)
		case "d":
			return int64(time.Hour * 24)
		default:
			return 1
		}
	}(), err
}

// parseAndValidPartition 解析并检验单测表达式的结果
func parseAndValidPartition(direction direction, val string, order ordered, control reflect.Value) (success bool, _ string) {
	var (
		rangeKey, target string
	)
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
		return order.ge(parseOrderNum(control, target)), "too low"
	case "(":
		return order.gt(parseOrderNum(control, target)), "too low"
	case "]":
		return order.le(parseOrderNum(control, target)), "too high"
	case ")":
		return order.lt(parseOrderNum(control, target)), "too high"
	default:
		panic("illegal border")
	}
	return
}

func doCheck(field, val string, control reflect.Value) error {
	valid := func(direction direction, singlePartition string) error {
		if succ, hint := parseAndValidPartition(direction, singlePartition, numOrderIns, control); !succ {
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

// replaceSelfExp 将 self.{字段名} 替换成具体值
func replaceSelfExp(str string, v reflect.Value) string {
	list := regMap[selfRegexp].FindAllString(str, -1)
	if len(list) == 0 {
		return str
	}

	for _, s := range list {
		name := strings.Trim(s, selfPlaceholder)
		value := getNonPtrValue(v.FieldByName(name))
		if !value.IsValid() {
			panic(fmt.Errorf("not found filed :%s", s))
		}
		//checkNumber(value, s)
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

func getNonPtrValue(value reflect.Value) reflect.Value {
	if value.Kind() == reflect.Pointer {
		return getNonPtrValue(value.Elem())
	}
	return value
}
func getExp(value reflect.Value) string {
	if value.Type() == durationType {
		return timeRegexp
	}
	return numRegexp
}

// Validate 校验结构体的字段是否合法
/*
- 使用 `valid` tag 进行标记
- 对于数字类型使用范围校验，通过 [] () 代表边界 ，数字和可以带符号 [+、-] , 可以使用 ~ 代表无穷
- 对于时间类型，范围校验与数组相同，需要在数字后面加上单位 [milli,m,h,d] ，不支持符号
- 也可以支持与当前结构体的其他字段进行比较，通过 self.{字段名} 指定 ，注意 int 和 time.Duration 也可以进行比较
*/
func Validate(i interface{}) (err error) {
	iType := getTypeStruct(reflect.TypeOf(i))
	iValue := getValueStruct(reflect.ValueOf(i))

	for idx := 0; idx < iType.NumField(); idx++ {
		var (
			field = iType.Field(idx)
			value = iValue.Field(idx)
		)
		// 结构体类型递归检查
		if v := getNonPtrValue(value); v.Kind() == reflect.Struct && v.CanInterface() {
			if err = Validate(value.Interface()); err != nil {
				return
			}
		}
		tag, ok := field.Tag.Lookup(validTag)
		if !ok {
			continue
		}
		// 检查字段类型是否符合要求
		if !checkFieldType(field) {
			panic(fmt.Sprintf("not support type :%s ,field :%s", field.Type, field.Name))
		}
		// 如果是空指针返回错误
		if value.IsNil() {
			return fmt.Errorf("field %s is nil", field.Name)
		}
		// 转换为具体类型
		value = getNonPtrValue(value)
		tag = replaceSelfExp(tag, iValue)
		checkTagValidate(getExp(value), tag)
		var (
			control reflect.Value
		)
		switch {
		case isNumKind[value.Kind()]: // 数值类型
			control = value
		case value.Kind() == reflect.String: // 字符串类型，复用数值类型的比较器
			control = reflect.ValueOf(value.Len())
		default:
			continue
		}
		if err = doCheck(field.Name, tag, control); err != nil {
			return err
		}
	}
	return nil
}

func checkFieldType(t reflect.StructField) bool {
	name := strings.ToLower(t.Type.String())
	check := func(substr ...string) bool {
		for _, s := range substr {
			if strings.Contains(name, s) {
				return true
			}
		}
		return false
	}
	return check("int", "float", "string")
}

func parseOrderNum(value reflect.Value, target string) (controlNum, targetNum interface{}) {
	var err error
	switch {
	case value.CanInt():
		targetNum, err = parseInt64(target)
		controlNum = value.Int()
	case value.CanUint():
		targetNum, err = strconv.ParseUint(target, 10, 64)
		controlNum = value.Uint()
	case value.CanFloat():
		targetNum, err = strconv.ParseFloat(target, 64)
		controlNum = value.Float()
	}
	if err != nil {
		panic("target parse number err")
	}
	return
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

	panic("getInt err")
}
