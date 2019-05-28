package network

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
)

func StructToString(v interface{}) string {
	return StructToStringIndent(v, 0)
}

func StructToStringIndent(vi interface{}, indent int) string {
	tabStr := strings.Repeat("\t", indent)
	str := ""
	t := reflect.TypeOf(vi).Elem()
	v := reflect.ValueOf(vi).Elem()
	for i := 0; i < t.NumField(); i++ {
		fv := v.Field(i)
		str += tabStr + t.Field(i).Name + ":"
		if t.Field(i).Type.Kind() == reflect.Struct {
			str += "\n" + StructToStringIndent(fv.Addr().Interface(), indent+1) + "\n"
		} else {
			str += fmt.Sprintf(" %v\n", fv.Interface())
		}
	}
	return str
}

type PriceGenerator struct {
	oldPrice    float64
	maxJump     float64
	upwardDrift float64
}

func NewPriceGenerator(startPrice, maxJump, upwardDrift float64) *PriceGenerator {
	return &PriceGenerator{
		oldPrice:    startPrice,
		maxJump:     maxJump,
		upwardDrift: upwardDrift,
	}
}

// old_price + (rand * 2 - 1)^2 * maximum_price_jump + upward_drift
func (pg *PriceGenerator) GenerateNewPrice() float64 {
	randomBits := rand.Float64()*2 - 1
	//fmt.Println("rand bits", randomBits)
	newPrice := pg.oldPrice + randomBits*pg.maxJump + pg.upwardDrift
	pg.oldPrice = newPrice
	return newPrice
}
