package main

import "io"

type getConfig struct {
	Client clientConfig

	ResourceType       string
	Names              []string
	QualifiedResources []string

	Output                   string
	NoHeaders                bool
	ShowManagedFields        bool
	Template                 string
	AllowMissingTemplateKeys bool
	ShowLabels               bool
	SortBy                   string
	LabelColumns             []string
	ShowKind                 bool

	Raw               string
	Watch             bool
	WatchOnly         bool
	OutputWatchEvents bool
	IgnoreNotFound    bool
	FieldSelector     string
	AllNamespaces     bool
	ServerPrint       bool
	Filenames         []string
	Kustomize         string
	Recursive         bool
	ChunkSize         int64
	LabelSelector     string
	Subresource       string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}
