# validate

- 用于校验结构体参数是否合法
- 支持类型 int 、uint、 float、time.Duration 、string
- 比较类型：范围（数字）、长度（字符串）


- 使用方式

```go
    type myInt int64
    type Param struct {
        A *int           `valid:"[-1,+3)"` // 在字段不为空的前提下，数值大于等于 -1 ，小于 3
        B float64       `valid:"[~,3.3)"` // 数值大小大于负无穷，小于 3.3
        C uint64        `valid:"[2,45435]"`
        D time.Duration `valid:"[500milli,3h]"` // 时间大于等于 500 毫秒，小于等于 3 小时
        E uint64
        F uint64 `valid:"[self.C,self.E]"`  // 数值大于等于 C 字段，小于等于 E 字段 ，三者类型必须相同
        G myInt  `valid:"[9,~)"`
        H string `valid:"[7,~)"` // 字符串长度大于 7 
    }
```


- 使用 `valid` tag 进行标记
- 对于数字类型使用范围校验，通过 [] () 代表边界 ，数字和可以带符号 [+、-] , 可以使用 ~ 代表无穷。
- 对于时间类型(仅指`time.Duration`)，范围校验与数字相同，需要在数字后面加上单位 [milli,m,h,d] ，不支持符号。
- 也可以支持与当前结构体的其他字段进行比较，通过 `self.{字段名}` 指定 ，注意 `int` 和 `time.Duration` 也可以进行比较
- `time.Duration` 如果别被替换成自定义类型则会当成 `int64` 进行解析
- 除 `time.Duration` 外的数值类型均支持自定义类型
- 对于字符串类型来说表达式规则和数字类型相同，但校验的字符串的长度


- api
```go
    Validate(i interface{}) error
```