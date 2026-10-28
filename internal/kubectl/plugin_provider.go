package kubectl

import (
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/plugin"
)

type PluginCatalog struct {
	root   []pluginNode
	create []pluginNode
}

type pluginNode struct {
	Name       string
	LookupName string
	Children   []pluginNode
}

func DiscoverPluginCatalog(searchPath string) PluginCatalog {
	options := &plugin.PluginListOptions{
		PluginPaths: filepath.SplitList(searchPath),
		IOStreams: genericiooptions.IOStreams{
			Out:    io.Discard,
			ErrOut: io.Discard,
		},
	}
	paths, _ := options.ListPlugins()

	seen := make(map[string]struct{})
	var catalog PluginCatalog
	for _, path := range paths {
		name := filepath.Base(path)
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}

		raw := strings.Split(name, "-")
		if len(raw) < 2 || raw[0] != "kubectl" {
			continue
		}
		raw = raw[1:]
		segments := make([]string, len(raw))
		valid := true
		for index, segment := range raw {
			segments[index] = strings.ReplaceAll(segment, "_", "-")
			if segments[index] == "" {
				valid = false
			}
		}
		if !valid {
			continue
		}

		catalog.root = addPluginPath(catalog.root, segments, strings.Join(raw, "-"))
		if segments[0] == "create" && len(segments) > 1 {
			catalog.create = addPluginPath(catalog.create, segments[1:], strings.Join(raw, "-"))
		}
	}
	return catalog
}

func addPluginPath(nodes []pluginNode, segments []string, lookupName string) []pluginNode {
	if len(segments) == 0 {
		return nodes
	}
	for index := range nodes {
		if nodes[index].Name != segments[0] {
			continue
		}
		if len(segments) == 1 {
			nodes[index].LookupName = lookupName
		} else {
			nodes[index].Children = addPluginPath(nodes[index].Children, segments[1:], lookupName)
		}
		return nodes
	}

	node := pluginNode{
		Name: segments[0],
	}
	if len(segments) == 1 {
		node.LookupName = lookupName
	} else {
		node.Children = addPluginPath(node.Children, segments[1:], lookupName)
	}
	return append(nodes, node)
}

func addPluginProviders[C any](subcommands *cli.Subcommands[C], nodes []pluginNode, occupied map[string]struct{}) {
	for _, node := range nodes {
		if _, collision := occupied[node.Name]; collision {
			continue
		}
		node := node
		subcommands.CommandsFrom(func(_ context.Context, batch *cli.CommandBatch[C]) error {
			batch.Command(pluginNodeCommand(node))
			return nil
		})
	}
}

func pluginNodeCommand(node pluginNode) cli.Command[externalPluginConfig] {
	command := cli.Command[externalPluginConfig]{
		Name: node.Name,
		Description: cli.Description{
			Short: "The command " + node.Name + " is a plugin installed by the user",
		},
		Declare: func(cb *cli.CommandBuilder[externalPluginConfig]) {
			for _, childNode := range node.Children {
				child := pluginNodeCommand(childNode)
				cb.Subcommands.Command(child)
			}
			if node.LookupName == "" {
				return
			}
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			remainder := cb.Arguments.Remainder(arguments, "ARG", cli.Count.ZeroOrMore()).HyphenValues(cli.HyphenValueAllow)
			remainder.DelegateCompletionBy(externalPluginCompletion(node.LookupName))
		},
	}
	if node.LookupName != "" {
		command.Run = runExternalPlugin(node.LookupName)
	}
	return command
}

func pluginOccupiedNames(names ...string) map[string]struct{} {
	occupied := make(map[string]struct{}, len(names))
	for _, name := range names {
		occupied[name] = struct{}{}
	}
	return occupied
}
