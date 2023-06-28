package githosts

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

func TestRenameInvalidBundle(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub test as GITHUB_TOKEN is missing")
	}

	// require.NoError(t, os.Setenv("")
	backupDir := os.Getenv(envVarGitBackupDir)
	dfDir := path.Join(backupDir, "github.com", "go-soba", "repo0")
	require.NoError(t, os.MkdirAll(dfDir, 0o755))
	dfName := "repo0.20200401111111.bundle"
	dfPath := path.Join(dfDir, dfName)
	_, err := os.OpenFile(dfPath, os.O_RDONLY|os.O_CREATE, 0o666)
	require.NoError(t, err)
	require.NoError(t, os.Setenv(githubEnvVarBackups, "1"))
	// run
	gh, err := NewGitHubHost(NewGitHubHostInput{
		APIURL:           githubAPIURL,
		DiffRemoteMethod: refsMethod,
		BackupDir:        backupDir,
		Token:            os.Getenv("GITHUB_TOKEN"),
	})
	require.NoError(t, err)

	gh.Backup()
	// check only one bundle remains
	files, err := os.ReadDir(dfDir)
	require.NoError(t, err)
	dfRenamed := "repo0.20200401111111.bundle.invalid"

	var originalFound int
	var renamedFound int
	for _, f := range files {
		require.NotEqual(t, f.Name(), dfName, fmt.Sprintf("unexpected bundle: %s", f.Name()))
		if dfName == f.Name() {
			originalFound++
		}

		if dfRenamed == f.Name() {
			renamedFound++
		}

	}
	require.Zero(t, originalFound)
	require.Equal(t, 1, renamedFound)
}
