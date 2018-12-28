package handlers

import (
	"html/template"
	"path"
	"path/filepath"
)

// renderer help use create Go template.Template easily and cache it. The template
// taken from function rather than from fs.
type renderer struct {
	asset          func(string) ([]byte, error)
	cachedTemplate map[string]*template.Template
}

func newTemplateRenderer(asset func(string) ([]byte, error)) *renderer {
	return &renderer{asset: asset, cachedTemplate: make(map[string]*template.Template)}
}

func (r *renderer) Template(name string, base string, tpls ...string) (tpl *template.Template, err error) {
	if tpl, ok := r.cachedTemplate[name]; ok {
		return tpl, nil
	}

	tpl = template.New(name)

	if tpl, err = r.parseTemplate(tpl, base); err != nil {
		return nil, err
	}

	for _, tn := range tpls {
		if tpl, err = r.parseTemplate(tpl, tn); err != nil {
			return nil, err
		}
	}

	r.cachedTemplate[name] = tpl

	return
}

func (r *renderer) parseTemplate(tpl *template.Template, name string) (*template.Template, error) {
	assetPath := path.Join("assets/templates", filepath.FromSlash(path.Clean("/"+name)))
	if len(assetPath) > 0 && assetPath[0] == '/' {
		assetPath = assetPath[1:]
	}

	var b []byte
	var err error
	if b, err = r.asset(assetPath); err != nil {
		return nil, err
	}

	return tpl.Parse(string(b))
}
