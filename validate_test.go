package validate

import (
	"math"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"
)

type TestCast struct {
	A int8          `valid:"[1,2]"`
	B time.Duration `valid:"[1,2]"`
}

func TestName(t *testing.T) {
	tc := TestCast{
		A: 1,
		B: time.Hour,
	}
	configType := reflect.TypeOf(tc)
	if configType.Kind() != reflect.Struct {
		panic("validate target must be struct")
	}
	configValue := reflect.ValueOf(tc)
	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)
		valid := field.Tag.Get("valid")
		if valid == "" {
			continue
		}
		t.Log(field.Type.String())
		value := configValue.Field(i)
		t.Log(value.Int())
	}
}

func TestReG(t *testing.T) {
	type testCase struct {
		str string
		res bool
	}
	testCaseList := []testCase{
		{
			str: "[ 4milli ,4milli]",
			res: true,
		},
		{
			str: "[1m,4m]",
			res: true,
		},

		{
			str: "[1h,4h]",
			res: true,
		},
		{
			str: "[1m,4d]",
			res: true,
		}, {
			str: "[1d,4d]",
			res: true,
		}, {
			str: "[1m,4d]",
			res: true,
		}, {
			str: "[-1m,4d]",
			res: false,
		}, {
			str: "[1m,+4d]",
			res: false,
		}, {
			str: "[1,4day]",
			res: false,
		}, {
			str: "[1milli,4d)",
			res: true,
		}, {
			str: "-[1milli,4d)",
			res: false,
		}, {
			str: "[1milli,4d)+",
			res: false,
		},
	}

	doCheck := func(exp string, testCaseList []testCase) {
		reg1 := regexp.MustCompile(exp)
		if reg1 == nil {
			panic("regexp validate")
		}
		for _, c := range testCaseList {
			//根据规则提取关键信息
			result := reg1.FindString(c.str)
			if len(result) > 0 != c.res {
				t.Fail()
				t.Logf("%s not true ,result :%v ", c.str, result)
			}
		}
	}
	doCheck(timeRegexp, testCaseList)
	doCheck(numRegexp, []testCase{
		{
			str: "[ 1,3]",
			res: true,
		},
		{
			str: "[2,4)",
			res: true,
		}, {
			str: "[-2,4)",
			res: true,
		}, {
			str: "[-2,+4)",
			res: true,
		}, {
			str: "-[-2,+4)",
			res: false,
		}, {
			str: "-[2,4)",
			res: false,
		}, {
			str: "-[2,4)+",
			res: false,
		}, {
			str: "[-,4)",
			res: false,
		}, {
			str: "[-,+)",
			res: false,
		}, {
			str: "[-,+4)",
			res: false,
		},
		{
			str: "[~,+4)",
			res: true,
		}, {
			str: "[4,~)",
			res: true,
		}, {
			str: "[~,~)",
			res: true,
		}, {
			str: "[~,~]",
			res: true,
		},
	})

}

func TestTimeReg(t *testing.T) {
	_ = `^[[(](\d+([mdh]|milli)){1},(\d+([mdh]|milli)){1}[])]$`
	self := `self\.\w+`
	reg1 := regexp.MustCompile(self)
	if reg1 == nil {
		panic("regexp validate")
	}
	result := reg1.FindAllString("[self.sDDDf,self.V]", -1)

	t.Log(result, len(result))
}

func TestReplace(t *testing.T) {
	type TestCase struct {
		A int
		B int
	}
	//v := reflect.ValueOf(TestCase{A: 1, B: 3})
	//t.Log(replaceSelfExp("[self.A,self.V]", v))

}

func TestParsePartition(t *testing.T) {
	t.Log(parsePartition(left, "[~", numCompare{}, "1"))
	t.Log(parsePartition(left, "[1", numCompare{}, "1"))
	t.Log(parsePartition(left, "(3", numCompare{}, "3"))
	t.Log(parsePartition(right, "4]", numCompare{}, "1"))
	t.Log(parsePartition(right, "3)", numCompare{}, "3"))
	t.Log(parsePartition(right, "~]", numCompare{}, "3"))

}

func TestString(t *testing.T) {
	right := "+33245]"
	right = string(append([]byte{right[len(right)-1]}, right[:len(right)-1]...))
	t.Log(right)
}

func TestGetStruct(t *testing.T) {
	type X struct {
	}
	var x = &X{}

	t.Log(getTypeStruct(reflect.TypeOf(x)).Kind())
	t.Log(getValueStruct(reflect.ValueOf(x)).Kind())
}

func TestValidate(t *testing.T) {
	type X struct {
		c *int32 `valid:"[1,3)"`
		//a int    `valid:"[1,~]"`
		b uint `valid:"[1,3]"`
		//d uint64 `valid:"[0,18446744073709551615]"`
		////e uint64 `valid:"[-3,18446744073709551615)"`
		//f int64         `valid:"[~,234234)"`
		//g time.Duration `valid:"[1milli,4m]"`
		////h time.Duration `valid:"[1h,4m]"`
		i int64   `valid:"[self.c,self.b]"`
		j float64 `valid:"[1.3,1.9]"`
		//k time.Duration `valid:"[1m,5m]"`
	}
	c := int32(2)
	var x = X{
		//a: 9,
		b: 3,
		c: &c,
		//d: 18446744073709551615,
		////e: 18446744073709551615,
		//f: 234234 - 1,
		//g: 4 * time.Minute,
		////h: time.Second,
		i: 3,
		j: 1.3,
	}

	t.Log(Validate(&x))
}

func TestToTimeDuration(t *testing.T) {
	t.Log(toTimeDuration("1min"))
	t.Log(toTimeDuration("1day"))
	t.Log(toTimeDuration("13milli"))
	t.Log(toTimeDuration("13900milli"))
}

func TestParserTimeDur(t *testing.T) {
	t.Log(parseTimeDuration("1m"))
	t.Log(parseTimeDuration("133mi"))
}

func TestGetInt(t *testing.T) {
	var i uint64 = math.MaxUint64
	var b uint64 = math.MaxUint64
	numCompare{}.lt(number2Str(reflect.ValueOf(i)), number2Str(reflect.ValueOf(b)))

}

func TestParseFloat(t *testing.T) {
	t.Log(strconv.FormatFloat(3.1415937, 'e', 10, 64))

}

type Param struct {
	A int           `valid:"[1,3)"`
	B float64       `valid:"[~,3.3)"`
	C uint64        `valid:"[2,45435]"`
	D time.Duration `valid:"[500milli,3h]"`
	E uint64
	F uint64 `valid:"[self.C,self.E]"`
}
