package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/proxy"
)

func runProxy(ctx context.Context, cfg *proxyConfig) error {
	clientConfig, err := cfg.Client.factoryForInvocation(ctx).ToRESTConfig()
	if err != nil {
		return err
	}

	if !cfg.AppendServerPath {
		target, err := url.Parse(clientConfig.Host)
		if err != nil {
			return err
		}
		if target.Path != "" && target.Path != "/" {
			klog.Warning("Your kube context contains a server path " + target.Path + ", use --append-server-path to automatically append the path to each request")
		}
	}

	var filter *proxy.FilterServer
	if cfg.DisableFilter {
		if cfg.UnixSocket == "" {
			klog.Warning("Request filter disabled, your proxy is vulnerable to XSRF attacks, please be cautious")
		}
	} else {
		if cfg.RejectPaths.Source == "" || cfg.RejectMethods.Source == "" {
			klog.Warning("Request filter has been relaxed, your proxy may be vulnerable to attacks, please be cautious")
		}
		filter = &proxy.FilterServer{
			AcceptPaths:   cfg.AcceptPaths.Expressions,
			RejectPaths:   cfg.RejectPaths.Expressions,
			AcceptHosts:   cfg.AcceptHosts.Expressions,
			RejectMethods: cfg.RejectMethods.Expressions,
		}
	}

	if cfg.Www != "" {
		info, err := os.Stat(cfg.Www)
		if err != nil {
			klog.Warning("Failed to stat static file directory "+cfg.Www+": ", err)
		} else if !info.IsDir() {
			klog.Warning("Static file directory " + cfg.Www + " is not a directory")
		}
	}

	server, err := proxy.NewServer(cfg.Www, cfg.APIPrefix, cfg.WwwPrefix, filter, clientConfig, cfg.Keepalive, cfg.AppendServerPath)
	if err != nil {
		return err
	}

	var listener net.Listener
	if cfg.UnixSocket == "" {
		listener, err = server.Listen(cfg.Address, cfg.Port)
	} else {
		listener, err = server.ListenUnix(cfg.UnixSocket)
	}
	if err != nil {
		return err
	}
	defer listener.Close()
	stop := context.AfterFunc(ctx, func() { _ = listener.Close() })
	defer stop()
	if _, err := fmt.Fprintf(cfg.Streams.Out, "Starting to serve on %s\n", listener.Addr().String()); err != nil {
		return err
	}
	if err := server.ServeOnListener(listener); err != nil {
		if cause := context.Cause(ctx); cause != nil {
			return cause
		}
		return err
	}
	return nil
}
