// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

// Code generated by code-gen/desenitize DO NOT EDIT

package desensitize

import (
	// "encoding/json"
	"cloudiac/portal/models"
)

type Variable struct {
	models.Variable
}

// 不定义 MarshalJSON() 方法，因为一旦定义了该结构体就无法组合使用了，
// 会覆盖 MarshalJSON() 方法以导致组合的其他字段不输出。 比如定义结构体:
//
//	type VariableWithExt struct {
//			models.Variable
//			Ext	string
//	}
//
// 当我们调用 json.Marshal(VariableWithExt{}) 时 Ext 字段不会输出，
// 因为直接调用了 models.Variable.MarshalJSON() 方法。
//
//	func (v Variable) MarshalJSON() ([]byte, error) {
//		return json.Marshal(v.Variable.Desensitize())
//	}
func (v Variable) Desensitize() Variable {
	return Variable{v.Variable.Desensitize()}
}

func NewVariable(v models.Variable) Variable {
	rv := Variable{v.Desensitize()}
	return rv
}

func NewVariablePtr(v *models.Variable) *Variable {
	rv := Variable{v.Desensitize()}
	return &rv
}

func NewVariableSlice(vs []models.Variable) []Variable {
	rvs := make([]Variable, len(vs))
	for i := 0; i < len(vs); i++ {
		rvs[i] = NewVariable(vs[i])
	}
	return rvs
}

func NewVariableSlicePtr(vs []*models.Variable) []*Variable {
	rvs := make([]*Variable, len(vs))
	for i := 0; i < len(vs); i++ {
		v := NewVariable(*vs[i])
		rvs[i] = &v
	}
	return rvs
}
