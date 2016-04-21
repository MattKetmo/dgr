package template

import (
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"io/ioutil"
	"os"
	"strings"
	txttmpl "text/template"
	"github.com/leekchan/gtf"
)

type TemplateDir struct {
	fields   data.Fields
	src      string
	dst      string
	partials *txttmpl.Template
}

func NewTemplateDir(path string, targetRoot string) (*TemplateDir, error) {
	fields := data.WithField("dir", path)
	logs.WithF(fields).Info("Reading template dir")
	tmplDir := &TemplateDir{
		fields: fields,
		src:    path,
		dst:    targetRoot,
	}
	return tmplDir, tmplDir.LoadPartial()
}

func (t *TemplateDir) LoadPartial() error {
	partials := []string{}

	directory, err := os.Open(t.src)
	if err != nil {
		return errs.WithEF(err, t.fields, "Failed to open template dir")
	}
	objects, err := directory.Readdir(-1)
	if err != nil {
		return errs.WithEF(err, t.fields, "Failed to read template dir")
	}
	for _, obj := range objects {
		if !obj.IsDir() && strings.HasSuffix(obj.Name(), ".partial") {
			partials = append(partials, t.src+"/"+obj.Name())
		}
	}

	if len(partials) == 0 {
		return nil
	}
	var tmpl *txttmpl.Template
	for _, partial := range partials {
		if tmpl == nil {
			tmpl = txttmpl.New(partial).Funcs(TemplateFunctions).Funcs(map[string]interface{}(gtf.GtfFuncMap))
		} else {
			tmpl = tmpl.New(partial).Funcs(TemplateFunctions).Funcs(map[string]interface{}(gtf.GtfFuncMap))
		}

		content, err := ioutil.ReadFile(partial)
		if err != nil {
			return errs.WithEF(err, t.fields.WithField("partial", partial), "Cannot read partial file")
		}
		tmpl, err = tmpl.Funcs(TemplateFunctions).Parse(CleanupOfTemplate(string(content)))
		if err != nil {
			return errs.WithEF(err, t.fields.WithField("partial", partial), "Failed to parse partial")
		}
	}
	t.partials = tmpl
	return nil
}

func (t *TemplateDir) Process(attributes map[string]interface{}) error {
	if err := t.processSingleDir(t.src, t.dst, attributes); err != nil {
		return err
	}
	return nil
}

func (t *TemplateDir) processSingleDir(src string, dst string, attributes map[string]interface{}) error {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return errs.WithEF(err, t.fields, "Cannot read source dir stat")
	}

	if err := os.MkdirAll(dst, sourceInfo.Mode()); err != nil {
		return errs.WithEF(err, t.fields, "Cannot create target directory templates")
	}

	directory, err := os.Open(src)
	if err != nil {
		return errs.WithEF(err, t.fields, "Cannot open directory")
	}
	objects, err := directory.Readdir(-1)
	if err != nil {
		return errs.WithEF(err, t.fields, "Cannot list files in directory")
	}
	for _, obj := range objects {
		srcObj := src + "/" + obj.Name()
		dstObj := dst + "/" + obj.Name()
		if obj.IsDir() {
			if err := t.processSingleDir(srcObj, dstObj, attributes); err != nil {
				return err
			}
		} else if strings.HasSuffix(obj.Name(), ".tmpl") {
			dstObj := dstObj[:len(dstObj)-5]
			template, err := NewTemplateFile(t.partials, srcObj, obj.Mode())
			if err != nil {
				return err
			}
			if err := template.runTemplate(dstObj, attributes); err != nil {
				return err
			}
		}
	}
	return nil
}
