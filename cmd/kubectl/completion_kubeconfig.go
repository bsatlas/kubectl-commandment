package main

import (
	"context"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

type kubeconfigNameKind uint8

const (
	kubeconfigContextName kubeconfigNameKind = iota + 1
	kubeconfigClusterName
	kubeconfigUserName
)

func kubeconfigNameCompleter[C any](path func(*C) string, kind kubeconfigNameKind) cli.ValueCompleter[C] {
	return func(_ context.Context, cfg *C, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		var (
			values        []string
			candidateKind = cli.CompletionCandidateValue
			err           error
		)
		switch kind {
		case kubeconfigContextName:
			values, err = contextNames(path(cfg))
		case kubeconfigClusterName:
			values, err = clusterNames(path(cfg))
		case kubeconfigUserName:
			values, err = userNames(path(cfg))
			candidateKind = cli.CompletionCandidateUser
		}
		if err != nil {
			return cli.CompletionContribution{}, err
		}
		filtered := values[:0]
		for _, value := range values {
			if strings.HasPrefix(value, request.Text) {
				filtered = append(filtered, value)
			}
		}
		return completionFromStrings(filtered, candidateKind, cli.CompletionSpacingDefault), nil
	}
}

func completeRootNamespace(ctx context.Context, cfg *rootConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	client := clientConfig{
		CacheDir:             cfg.CacheDir,
		Kubeconfig:           cfg.Kubeconfig,
		Cluster:              cfg.Cluster,
		User:                 cfg.User,
		Context:              cfg.Context,
		Server:               cfg.Server,
		TLSServerName:        cfg.TLSServerName,
		Insecure:             cfg.Insecure,
		ClientCertificate:    cfg.ClientCertificate,
		ClientKey:            cfg.ClientKey,
		CertificateAuthority: cfg.CertificateAuthority,
		Token:                cfg.Token,
		Impersonate:          cfg.Impersonate,
		ImpersonateUID:       cfg.ImpersonateUID,
		ImpersonateGroups:    cfg.ImpersonateGroups,
		ImpersonateUserExtra: cfg.ImpersonateUserExtra,
		Username:             cfg.Username,
		Password:             cfg.Password,
		RequestTimeout:       cfg.RequestTimeout,
		DisableCompression:   cfg.DisableCompression,
		ProxyURL:             cfg.ProxyURL,
	}
	completion, err := completeResourceNames(ctx, restClientGetter(client), "namespace", resourceCompletionBare)
	if err != nil {
		return cli.CompletionContribution{}, err
	}
	completion.Candidates = filterCompletionCandidates(completion.Candidates, request.Text, nil)
	for index := range completion.Candidates {
		completion.Candidates[index].Kind = cli.CompletionCandidateNamespace
	}
	return completion, nil
}
