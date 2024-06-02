package manifest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"net/http"
	"text/template"

	"github.com/okteto/okteto/cmd/utils"
	oktetoLog "github.com/okteto/okteto/pkg/log"
	"github.com/okteto/okteto/pkg/model"
	"github.com/okteto/okteto/pkg/okteto"
	"gopkg.in/yaml.v3"
)

func (*Command) RunCreateTemplate(ctx context.Context, opts *InitOpts) (*model.Manifest, error) {
	var tmplCnt string
	var path string
	if opts.DevPath != "" {
		path = opts.DevPath
	} else {
		path = filepath.Join(opts.Workdir, utils.DefaultManifest)
	}

	if err := validateDevPath(path, opts.Overwrite); err != nil {
		return nil, err
	}

	tmplCnt, err := getUriContents(opts.Template)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("okteto manifest").Parse(tmplCnt)
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer

	ns := opts.Namespace
	ctxName := opts.Context
	if ns == "" {
		ns = okteto.GetContext().Namespace
	}
	if ctxName == "" {
		ctxName = okteto.GetContext().Name
	}
	if opts.TemplateArgs == nil {
		opts.TemplateArgs = map[string]string{}
	}
	opts.TemplateArgs["namespace"] = ns
	opts.TemplateArgs["context"] = ctxName
	targs := map[string]interface{}{}
	if opts.TemplateArgFile != "" {
		targs, err = loadArgumentFiles(opts.TemplateArgFile)
		if err != nil {
			return nil, err
		}
	}
	// load inline arguments
	for k, v := range opts.TemplateArgs {
		targs[k] = v
	}

	err = tmpl.Execute(&buffer, targs)
	if err != nil {
		return nil, err
	}
	oktetoLog.Success("Generated template successfully.")
	m, err := model.Read(buffer.Bytes())
	if err != nil {
		return nil, err
	}
	return m, nil
}

func getUriContents(tmplUri string) (string, error) {
	var tmplCnt string
	if isHttpUrl(tmplUri) {
		resp, err := http.Get(tmplUri)
		defer resp.Body.Close()
		if err != nil {
			return "", err
		}

		// Check for non-200 status code
		if resp.StatusCode != http.StatusOK {
			return "", errors.New(fmt.Sprintf("failed to fetch URL: %s, status code: %d", tmplUri, resp.StatusCode))
		}

		bytes, err := io.ReadAll(resp.Body)

		if err != nil {
			return "", err
		}

		tmplCnt = string(bytes)
	} else {
		bytes, err := os.ReadFile(tmplUri)
		if err != nil {
			return "", err
		}

		tmplCnt = string(bytes)
	}

	return tmplCnt, nil
}

func isHttpUrl(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}

func loadArgumentFiles(path string) (map[string]interface{}, error) {
	argsMap := map[string]interface{}{}
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(file, &argsMap)
	if err != nil {
		return nil, err
	}
	return argsMap, nil
}
