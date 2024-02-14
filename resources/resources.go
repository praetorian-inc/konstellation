package resources

import (
	"embed"
	_ "embed"
)

//go:embed k8s/*
var K8sConfigPath embed.FS

//go:embed k8s/config.yml
var K8sConfigFile []byte
