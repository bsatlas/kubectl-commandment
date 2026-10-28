package main

import (
	"fmt"
	"math"

	"k8s.io/apimachinery/pkg/util/version"
)

func versionSkewWarning(clientGitVersion, serverGitVersion string) (string, error) {
	clientVersion, err := version.ParseSemantic(clientGitVersion)
	if err != nil {
		return "", fmt.Errorf("client version error: %w", err)
	}
	serverVersion, err := version.ParseSemantic(serverGitVersion)
	if err != nil {
		return "", fmt.Errorf("server version error: %w", err)
	}

	majorDifference := math.Abs(float64(clientVersion.Major()) - float64(serverVersion.Major()))
	minorDifference := math.Abs(float64(clientVersion.Minor()) - float64(serverVersion.Minor()))
	if majorDifference > 0 || minorDifference > 1 {
		return fmt.Sprintf(
			"version difference between client (%d.%d) and server (%d.%d) exceeds the supported minor version skew of +/-1",
			clientVersion.Major(), clientVersion.Minor(), serverVersion.Major(), serverVersion.Minor(),
		), nil
	}
	return "", nil
}
