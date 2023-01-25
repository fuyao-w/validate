# validate

- 用于校验结构体参数是否合法
- 支持类型 int 、uint、 float、time.Duration
- 比较类型：范围


- 使用方式

```go
    type Param struct {
        A int           `valid:"[-1,+3)"`
        B float64       `valid:"[~,3.3)"`
        C uint64        `valid:"[2,45435]"`
        D time.Duration `valid:"[500milli,3h]"`
        E uint64
        F uint64 `valid:"[self.C,self.E]"`
    }
```


- 使用 valid tag 进行标记
- 对于数字类型使用范围校验，通过 [] () 代表边界 ，数字和可以带符号 [+、-] , 可以使用 ~ 代表无穷
- 对于时间类型，范围校验与数字相同，需要在数字后面加上单位 [milli,m,h,d] ，不支持符号
- 也可以支持与当前结构体的其他字段进行比较，通过 self.{字段名} 指定 ，注意 int 和 time.Duration 也可以进行比较


- api
```go
    Validate(i interface{}) error
```