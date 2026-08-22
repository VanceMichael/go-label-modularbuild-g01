package complianceexport

import (
	"encoding/csv"
	"fmt"
	"io"
)

type Encoder struct {
	csv *csv.Writer
}

func NewEncoder(writer io.Writer) *Encoder {
	return &Encoder{csv: csv.NewWriter(writer)}
}

func (e *Encoder) Header() error {
	return e.csv.Write([]string{"module_id", "serial", "site_code", "signature"})
}

func (e *Encoder) Module(module Module, signature string) error {
	if signature == "" {
		return fmt.Errorf("compliance export: empty signature")
	}
	return e.csv.Write([]string{module.ID, module.Serial, module.SiteCode, signature})
}

func (e *Encoder) Flush() error {
	e.csv.Flush()
	return e.csv.Error()
}
