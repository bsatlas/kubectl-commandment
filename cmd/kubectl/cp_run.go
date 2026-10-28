/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/exec"
)

type copyPathSpec interface {
	String() string
}

type copyLocalPath struct {
	file string
}

func newCopyLocalPath(fileName string) copyLocalPath {
	file := stripCopyTrailingSlash(fileName)
	path := copyLocalPath{
		file: file,
	}
	return path
}

func (p copyLocalPath) String() string { return p.file }

func (p copyLocalPath) Dir() copyLocalPath { return newCopyLocalPath(filepath.Dir(p.file)) }

func (p copyLocalPath) Base() copyLocalPath { return newCopyLocalPath(filepath.Base(p.file)) }

func (p copyLocalPath) Clean() copyLocalPath { return newCopyLocalPath(filepath.Clean(p.file)) }

func (p copyLocalPath) Join(elem copyPathSpec) copyLocalPath {
	return newCopyLocalPath(filepath.Join(p.file, elem.String()))
}

func (p copyLocalPath) Glob() ([]string, error) { return filepath.Glob(p.file) }

func copyPathIsRelative(base, target copyLocalPath) bool {
	baseNormalized := strings.ReplaceAll(stripCopyTrailingSlash(base.String()), `\`, "/")
	targetNormalized := strings.ReplaceAll(stripCopyTrailingSlash(target.String()), `\`, "/")
	relative, err := filepath.Rel(baseNormalized, targetNormalized)
	if err != nil {
		return false
	}
	return relative == "." || relative == stripCopyPathShortcuts(relative)
}

type copyRemotePath struct {
	file string
}

func newCopyRemotePath(fileName string) copyRemotePath {
	file := strings.ReplaceAll(stripCopyTrailingSlash(fileName), `\`, "/")
	path := copyRemotePath{
		file: file,
	}
	return path
}

func (p copyRemotePath) String() string { return p.file }

func (p copyRemotePath) Dir() copyRemotePath { return newCopyRemotePath(path.Dir(p.file)) }

func (p copyRemotePath) Base() copyRemotePath { return newCopyRemotePath(path.Base(p.file)) }

func (p copyRemotePath) Clean() copyRemotePath { return newCopyRemotePath(path.Clean(p.file)) }

func (p copyRemotePath) Join(elem copyPathSpec) copyRemotePath {
	return newCopyRemotePath(path.Join(p.file, elem.String()))
}

func (p copyRemotePath) StripSlashes() copyRemotePath {
	return newCopyRemotePath(stripCopyLeadingSlash(p.file))
}

func stripCopyTrailingSlash(file string) string {
	if file == "" || file == "/" {
		return file
	}
	if strings.HasSuffix(file, "/") {
		return file[:len(file)-1]
	}
	return file
}

func stripCopyLeadingSlash(file string) string { return strings.TrimLeft(file, `/\`) }

func stripCopyPathShortcuts(value string) string {
	result := value
	for {
		trimmed := strings.TrimPrefix(result, "../")
		if trimmed == result {
			break
		}
		result = trimmed
	}
	for {
		trimmed := strings.TrimPrefix(result, `..\`)
		if trimmed == result {
			break
		}
		result = trimmed
	}
	if result == "." || result == ".." {
		result = ""
	}
	if result != "" && (result[0] == '/' || result[0] == '\\') {
		return result[1:]
	}
	return result
}

type copyRuntime struct {
	ctx        context.Context
	container  string
	namespace  string
	noPreserve bool
	maxTries   int

	clientConfig *rest.Config
	clientset    kubernetes.Interface
	executor     exec.RemoteExecutor
	streams      genericiooptions.IOStreams
}

func runCopy(ctx context.Context, cfg *cpConfig) error {
	source := cfg.Source
	destination := cfg.Destination

	factory := cfg.Client.factory()
	namespace, _, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	clientConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}

	operation := copyRuntime{
		ctx:          ctx,
		container:    cfg.Container,
		namespace:    namespace,
		noPreserve:   cfg.NoPreserve,
		maxTries:     cfg.Retries,
		clientConfig: clientConfig,
		clientset:    clientset,
		executor:     &exec.DefaultRemoteExecutor{},
		streams:      cfg.Streams,
	}
	if source.PodName != "" {
		return operation.copyFromPod(source, destination)
	}
	return operation.copyToPod(source, destination, &exec.ExecOptions{})
}

func (o *copyRuntime) checkDestinationIsDir(destination copyFileSpec) error {
	options := &exec.ExecOptions{
		StreamOptions: exec.StreamOptions{
			IOStreams: genericiooptions.IOStreams{
				Out:    io.Discard,
				ErrOut: io.Discard,
			},
			Namespace: destination.PodNamespace,
			PodName:   destination.PodName,
		},
		Command:  []string{"test", "-d", destination.File},
		Executor: o.executor,
	}

	ctx, cancel := context.WithTimeout(o.ctx, 30*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- o.execute(options) }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (o *copyRuntime) copyToPod(source, destination copyFileSpec, options *exec.ExecOptions) error {
	if _, err := os.Stat(source.File); err != nil {
		return fmt.Errorf("%s doesn't exist in local filesystem", source.File)
	}
	reader, writer := io.Pipe()

	sourceFile := newCopyLocalPath(source.File)
	destinationFile := newCopyRemotePath(destination.File)
	if err := o.checkDestinationIsDir(destination); err == nil {
		destinationFile = destinationFile.Join(sourceFile.Base())
	} else if errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	go func() {
		err := makeCopyTar(sourceFile, destinationFile, writer)
		_ = writer.CloseWithError(err)
	}()

	command := []string{"tar", "-xmf", "-"}
	if o.noPreserve {
		command = []string{"tar", "--no-same-permissions", "--no-same-owner", "-xmf", "-"}
	}
	if directory := destinationFile.Dir().String(); directory != "" {
		command = append(command, "-C", directory)
	}

	options.StreamOptions = exec.StreamOptions{
		IOStreams: genericiooptions.IOStreams{
			In:     reader,
			Out:    o.streams.Out,
			ErrOut: o.streams.ErrOut,
		},
		Stdin:     true,
		Namespace: destination.PodNamespace,
		PodName:   destination.PodName,
	}
	options.Command = command
	options.Executor = o.executor
	return o.execute(options)
}

func (o *copyRuntime) copyFromPod(source, destination copyFileSpec) error {
	reader := newCopyTarPipe(source, o)
	sourceFile := newCopyRemotePath(source.File)
	destinationFile := newCopyLocalPath(destination.File)
	prefix := stripCopyPathShortcuts(sourceFile.StripSlashes().Clean().String())
	return o.untarAll(source.PodNamespace, source.PodName, prefix, sourceFile, destinationFile, reader)
}

type copyTarPipe struct {
	source    copyFileSpec
	operation *copyRuntime
	reader    *io.PipeReader
	writer    *io.PipeWriter
	bytesRead uint64
	retries   int
}

func newCopyTarPipe(source copyFileSpec, operation *copyRuntime) *copyTarPipe {
	pipe := &copyTarPipe{
		source:    source,
		operation: operation,
	}
	pipe.initReadFrom(0)
	return pipe
}

func (pipe *copyTarPipe) initReadFrom(offset uint64) {
	pipe.reader, pipe.writer = io.Pipe()
	options := &exec.ExecOptions{
		StreamOptions: exec.StreamOptions{
			IOStreams: genericiooptions.IOStreams{
				Out:    pipe.writer,
				ErrOut: pipe.operation.streams.Out,
			},
			Namespace: pipe.source.PodNamespace,
			PodName:   pipe.source.PodName,
		},
		Command:  []string{"tar", "cf", "-", pipe.source.File},
		Executor: pipe.operation.executor,
	}
	if pipe.operation.maxTries != 0 {
		escapedPath := strings.ReplaceAll(pipe.source.File, "'", `'\''`)
		options.Command = []string{"sh", "-c", fmt.Sprintf("tar cf - '%s' | tail -c+%d", escapedPath, offset)}
	}
	go func() {
		err := pipe.operation.execute(options)
		_ = pipe.writer.CloseWithError(err)
	}()
}

func (pipe *copyTarPipe) Read(buffer []byte) (int, error) {
	count, err := pipe.reader.Read(buffer)
	if err == nil {
		pipe.bytesRead += uint64(count)
		return count, nil
	}
	if pipe.operation.maxTries < 0 || pipe.retries < pipe.operation.maxTries {
		pipe.retries++
		fmt.Fprintf(pipe.operation.streams.Out, "Resuming copy at %d bytes, retry %d/%d\n", pipe.bytesRead, pipe.retries, pipe.operation.maxTries)
		pipe.initReadFrom(pipe.bytesRead + 1)
		return count, nil
	}
	fmt.Fprintf(pipe.operation.streams.Out, "Dropping out copy after %d retries\n", pipe.retries)
	return count, err
}

func makeCopyTar(source copyLocalPath, destination copyRemotePath, writer io.Writer) error {
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()
	sourcePath := source.Clean()
	destinationPath := destination.Clean()
	return recursiveCopyTar(sourcePath.Dir(), sourcePath.Base(), destinationPath.Dir(), destinationPath.Base(), tarWriter)
}

func recursiveCopyTar(sourceDirectory, sourceFile copyLocalPath, destinationDirectory, destinationFile copyRemotePath, writer *tar.Writer) error {
	matches, err := sourceDirectory.Join(sourceFile).Glob()
	if err != nil {
		return err
	}
	for _, match := range matches {
		stat, err := os.Lstat(match)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			files, err := os.ReadDir(match)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				header, err := tar.FileInfoHeader(stat, match)
				if err != nil {
					return err
				}
				header.Name = destinationFile.String()
				if err := writer.WriteHeader(header); err != nil {
					return err
				}
			}
			for _, file := range files {
				if err := recursiveCopyTar(
					sourceDirectory,
					sourceFile.Join(newCopyLocalPath(file.Name())),
					destinationDirectory,
					destinationFile.Join(newCopyRemotePath(file.Name())),
					writer,
				); err != nil {
					return err
				}
			}
			return nil
		}
		if stat.Mode()&os.ModeSymlink != 0 {
			header, err := tar.FileInfoHeader(stat, match)
			if err != nil {
				return err
			}
			target, err := os.Readlink(match)
			if err != nil {
				return err
			}
			header.Linkname = target
			header.Name = destinationFile.String()
			return writer.WriteHeader(header)
		}

		header, err := tar.FileInfoHeader(stat, match)
		if err != nil {
			return err
		}
		header.Name = destinationFile.String()
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		file, err := os.Open(match)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	return nil
}

func (o *copyRuntime) untarAll(namespace, pod, prefix string, source copyRemotePath, destination copyLocalPath, reader io.Reader) error {
	symlinkWarningPrinted := false
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if !strings.HasPrefix(header.Name, prefix) {
			return fmt.Errorf("tar contents corrupted")
		}

		destinationFile := destination.Join(newCopyRemotePath(header.Name[len(prefix):]))
		if !copyPathIsRelative(destination, destinationFile) {
			fmt.Fprintf(o.streams.ErrOut, "warning: file %q is outside target destination, skipping\n", destinationFile)
			continue
		}
		if err := os.MkdirAll(destinationFile.Dir().String(), 0o755); err != nil {
			return err
		}
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(destinationFile.String(), 0o755); err != nil {
				return err
			}
			continue
		}
		if header.FileInfo().Mode()&os.ModeSymlink != 0 {
			if !symlinkWarningPrinted {
				fmt.Fprintf(o.streams.ErrOut,
					"warning: skipping symlink: %q -> %q (consider using \"kubectl exec -n %q %q -- tar cf - %q | tar xf -\")\n",
					destinationFile, header.Linkname, namespace, pod, source)
				symlinkWarningPrinted = true
				continue
			}
			fmt.Fprintf(o.streams.ErrOut, "warning: skipping symlink: %q -> %q\n", destinationFile, header.Linkname)
			continue
		}

		file, err := os.Create(destinationFile.String())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(file, tarReader)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
}

func (o *copyRuntime) execute(options *exec.ExecOptions) error {
	if options.Namespace == "" {
		options.Namespace = o.namespace
	}
	if o.container != "" {
		options.ContainerName = o.container
	}
	options.Config = o.clientConfig
	options.PodClient = o.clientset.CoreV1()
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}
