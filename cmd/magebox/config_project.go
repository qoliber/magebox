package main

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/docker"
)

type projectEndpoint struct {
	Service string `json:"service"`
	Version string `json:"version,omitempty"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
}

type projectDomain struct {
	Host      string `json:"host"`
	Root      string `json:"root"`
	SSL       bool   `json:"ssl"`
	StoreCode string `json:"store_code"`
	StoreType string `json:"store_type"`
}

type projectConfigJSON struct {
	Config  map[string]interface{} `json:"config"`
	Derived struct {
		DatabaseName string           `json:"database_name"`
		Database     *projectEndpoint `json:"database"`
		Cache        *projectEndpoint `json:"cache"`
		Search       *projectEndpoint `json:"search"`
		Domains      []projectDomain  `json:"domains"`
	} `json:"derived"`
}

func isProjectConfigJSON(cmd *cobra.Command) bool {
	if cmd.Name() != "config" || cmd.Flags().Lookup("json") == nil {
		return false
	}
	jsonOutput, _ := cmd.Flags().GetBool("json")
	return jsonOutput
}

func runConfigProject(cmd *cobra.Command, args []string) error {
	if !isProjectConfigJSON(cmd) {
		return cmd.Help()
	}
	cwd, err := getCwd()
	if err != nil {
		return err
	}
	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		return err
	}

	// Use the config's YAML representation to retain its existing field names
	// and custom service and command representations.
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	var result projectConfigJSON
	if err := yaml.Unmarshal(data, &result.Config); err != nil {
		return err
	}

	result.Derived.DatabaseName = cfg.DatabaseName()
	if db, err := getDbInfo(cfg); err == nil {
		result.Derived.Database = &projectEndpoint{Service: db.Type, Version: db.Version, Host: "127.0.0.1", Port: db.Port}
	}
	if cfg.Services.HasCacheService() {
		name := cfg.Services.GetCacheServiceName()
		svc := cfg.Services.Redis
		if name == "valkey" {
			svc = cfg.Services.Valkey
		}
		result.Derived.Cache = &projectEndpoint{Service: name, Version: svc.Version, Host: "127.0.0.1", Port: RedisDefaultPort}
	}
	if cfg.Services.HasOpenSearch() {
		result.Derived.Search = &projectEndpoint{Service: "opensearch", Version: cfg.Services.OpenSearch.Version, Host: "127.0.0.1", Port: docker.GetOpenSearchPort()}
	} else if cfg.Services.HasElasticsearch() {
		result.Derived.Search = &projectEndpoint{Service: "elasticsearch", Version: cfg.Services.Elasticsearch.Version, Host: "127.0.0.1", Port: docker.GetElasticsearchPort()}
	}
	for _, d := range cfg.Domains {
		result.Derived.Domains = append(result.Derived.Domains, projectDomain{
			Host: d.Host, Root: d.GetRootForProject(cwd, cfg.GetType()), SSL: d.IsSSLEnabled(),
			StoreCode: d.StoreCode, StoreType: d.GetStoreType(),
		})
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
