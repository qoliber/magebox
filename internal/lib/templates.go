// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package lib

import (
	"fmt"
	"os"
	"sync"
	"text/template"
)

// TemplateCategory constants for template organization
const (
	TemplateNginx   = "nginx"
	TemplatePHP     = "php"
	TemplateVarnish = "varnish"
	TemplateDNS     = "dns"
	TemplateSSL     = "ssl"
	TemplateXdebug  = "xdebug"
	TemplateProject = "project"
	TemplateWrapper = "wrappers"
)

// Global template loader singleton
var (
	globalLoader     *TemplateLoader
	globalLoaderOnce sync.Once
	globalLoaderMu   sync.RWMutex
)

// TemplateLoader loads templates from the lib directory with fallback support
type TemplateLoader struct {
	paths    *Paths
	fallback map[string]string // category/name -> embedded content
}

// NewTemplateLoader creates a new template loader
func NewTemplateLoader(paths *Paths) *TemplateLoader {
	return &TemplateLoader{
		paths:    paths,
		fallback: make(map[string]string),
	}
}

// RegisterFallback registers an embedded template as fallback
// This is used when the lib directory doesn't exist or the template is missing
func (t *TemplateLoader) RegisterFallback(category, name, content string) {
	key := category + "/" + name
	t.fallback[key] = content
}

// LoadTemplate loads a template by category and name
// It first checks the lib directory, then falls back to embedded templates
func (t *TemplateLoader) LoadTemplate(category, name string) (string, error) {
	// Try to load from lib directory first
	if t.paths != nil && t.paths.Exists() {
		templatePath := t.paths.TemplatePath(category, name)
		if data, err := os.ReadFile(templatePath); err == nil {
			return string(data), nil
		}
	}

	// Fall back to embedded template
	key := category + "/" + name
	if content, ok := t.fallback[key]; ok {
		return content, nil
	}

	return "", fmt.Errorf("template not found: %s/%s", category, name)
}

// LoadAndParseTemplate loads and parses a template
func (t *TemplateLoader) LoadAndParseTemplate(category, name string) (*template.Template, error) {
	content, err := t.LoadTemplate(category, name)
	if err != nil {
		return nil, err
	}

	return template.New(name).Parse(content)
}

// LoadRawFile loads a non-template file (like shell scripts) from lib
func (t *TemplateLoader) LoadRawFile(category, name string) (string, error) {
	return t.LoadTemplate(category, name)
}

// DefaultTemplateLoader creates a template loader with default paths
func DefaultTemplateLoader() (*TemplateLoader, error) {
	paths, err := DefaultPaths()
	if err != nil {
		return nil, err
	}
	return NewTemplateLoader(paths), nil
}

// TemplateNames returns all known template names organized by category
var TemplateNames = map[string][]string{
	TemplateNginx: {
		"vhost.conf.tmpl",
		"proxy.conf.tmpl",
		"upstream.conf.tmpl",
	},
	TemplatePHP: {
		"pool.conf.tmpl",
		"php-fpm.conf.tmpl",
		"not-installed-message.tmpl",
		"mailpit-sendmail.sh",
	},
	TemplateVarnish: {
		"default.vcl.tmpl",
	},
	TemplateDNS: {
		"dnsmasq.conf.tmpl",
		"systemd-resolved.conf.tmpl",
		"hosts-section.tmpl",
	},
	TemplateSSL: {
		"not-installed-error.tmpl",
	},
	TemplateXdebug: {
		"xdebug.ini.tmpl",
	},
	TemplateProject: {
		"env.php.tmpl",
	},
	TemplateWrapper: {
		"php.sh",
		"composer.sh",
		"blackfire.sh",
	},
}

// GetGlobalLoader returns the global template loader, initializing it if needed
func GetGlobalLoader() *TemplateLoader {
	globalLoaderOnce.Do(func() {
		paths, _ := DefaultPaths()
		globalLoader = NewTemplateLoader(paths)
	})
	return globalLoader
}

// RegisterFallbackTemplate registers an embedded template as fallback globally
// This should be called during package init() to register embedded templates
func RegisterFallbackTemplate(category, name, content string) {
	loader := GetGlobalLoader()
	globalLoaderMu.Lock()
	defer globalLoaderMu.Unlock()
	loader.RegisterFallback(category, name, content)
}

// GetTemplate loads a template from lib or falls back to embedded
// This is the main function packages should use to get templates
func GetTemplate(category, name string) (string, error) {
	return GetGlobalLoader().LoadTemplate(category, name)
}

// GetParsedTemplate loads and parses a template
func GetParsedTemplate(category, name string) (*template.Template, error) {
	return GetGlobalLoader().LoadAndParseTemplate(category, name)
}

// MustGetTemplate loads a template, panics if not found
func MustGetTemplate(category, name string) string {
	content, err := GetTemplate(category, name)
	if err != nil {
		panic(err)
	}
	return content
}
