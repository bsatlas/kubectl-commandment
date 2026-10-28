package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/proxy"
)

func runProxy(_ context.Context, cfg *proxyConfig) error {
	clientConfig, err := cfg.Client.factory().ToRESTConfig()
	if err != nil {
		return err
	}

	staticPrefix := cfg.WwwPrefix
	if !strings.HasSuffix(staticPrefix, "/") {
		staticPrefix += "/"
	}
	apiPrefix := cfg.APIPrefix
	if !strings.HasSuffix(apiPrefix, "/") {
		apiPrefix += "/"
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
		if cfg.RejectPaths == "" || cfg.RejectMethods == "" {
			klog.Warning("Request filter has been relaxed, your proxy may be vulnerable to attacks, please be cautious")
		}
		filter = &proxy.FilterServer{
			AcceptPaths:   proxy.MakeRegexpArrayOrDie(cfg.AcceptPaths),
			RejectPaths:   proxy.MakeRegexpArrayOrDie(cfg.RejectPaths),
			AcceptHosts:   proxy.MakeRegexpArrayOrDie(cfg.AcceptHosts),
			RejectMethods: proxy.MakeRegexpArrayOrDie(cfg.RejectMethods),
		}
	}

	if cfg.Port != 8001 && cfg.UnixSocket != "" {
		return fmt.Errorf("cannot set --unix-socket and --port at the same time")
	}
	if cfg.Www != "" {
		info, err := os.Stat(cfg.Www)
		if err != nil {
			klog.Warning("Failed to stat static file directory "+cfg.Www+": ", err)
		} else if !info.IsDir() {
			klog.Warning("Static file directory " + cfg.Www + " is not a directory")
		}
	}

	server, err := proxy.NewServer(cfg.Www, apiPrefix, staticPrefix, filter, clientConfig, cfg.Keepalive, cfg.AppendServerPath)
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
	fmt.Fprintf(cfg.Streams.Out, "Starting to serve on %s\n", listener.Addr().String())
	return server.ServeOnListener(listener)
}
