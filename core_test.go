package githosts

import (
	b64 "encoding/base64"
	"fmt"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func deleteBackupsDir(path string) error {
	return os.RemoveAll(path)
}

func createTestTextFile(fileName, content string) string {
	tmpDir := os.TempDir()
	dir, err := os.MkdirTemp(tmpDir, "soba-*")
	if err != nil {
		panic(err)
	}

	f, err := os.Create(filepath.Join(dir, fileName))
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		log.Println(err)
	}

	return filepath.Clean(f.Name())
}

func TestHostsImplementGitHostsInterface(t *testing.T) {
	require.Implements(t, (*gitProvider)(nil), new(GiteaHost))
	require.Implements(t, (*gitProvider)(nil), new(GitHubHost))
	require.Implements(t, (*gitProvider)(nil), new(BitbucketHost))
	require.Implements(t, (*gitProvider)(nil), new(GitlabHost))
}

func TestGetLatestBundleRefs(t *testing.T) {
	refs, err := getLatestBundleRefs("testfiles/example-bundles")
	require.NoError(t, err)
	var found int
	for k, v := range refs {
		switch k {
		case "refs/heads/master":
			if v == "2c84a508078d81eae0246ae3f3097befd0bcb7dc" {
				found++
			}
		case "refs/heads/my-branch":
			if v == "e16f7204b7640723bafc020c78ab29f4ea9f9265" {
				found++
			}
		case "HEAD":
			if v == "2c84a508078d81eae0246ae3f3097befd0bcb7dc" {
				found++
			}
		}
	}
}

func TestGetSHA2Hash(t *testing.T) {
	pathOne := createTestTextFile("one", "some content")
	sha, err := getSHA2Hash(pathOne)
	require.NoError(t, err)

	expectedSHA := "KQ9JPET11j0Gs3TQpavSkvrji5LKsvrl7+/hsOk0f1Y="
	require.Equal(t, expectedSHA, b64.StdEncoding.EncodeToString(sha))

	pathTwo := createTestTextFile("one", "some more content")
	sha, err = getSHA2Hash(pathTwo)
	require.NoError(t, err)
	require.NotEqual(t, expectedSHA, b64.StdEncoding.EncodeToString(sha))

	sha, err = getSHA2Hash("missing-file")
	require.Error(t, err)
	require.Empty(t, sha)
	require.Contains(t, err.Error(), "failed to open file")
}

func TestFilesIdentical(t *testing.T) {
	pathOne := createTestTextFile("one", "some content")
	pathTwo := createTestTextFile("two", "some content")
	require.True(t, filesIdentical(pathOne, pathTwo))

	pathOne = createTestTextFile("one", "some content")
	pathTwo = createTestTextFile("two", "some other content")
	require.False(t, filesIdentical(pathOne, pathTwo))
}

func TestGetTimeStampPartFromFileName(t *testing.T) {
	// test success
	timeStamp, err := getTimeStampPartFromFileName("repoName.20221102153359.bundle")
	require.NoError(t, err)
	require.Equal(t, 20221102153359, timeStamp)

	// test invalid format without enough tokens
	timeStamp, err = getTimeStampPartFromFileName("repoName.20221102153359")
	require.Error(t, err)
	require.Contains(t, err.Error(), "bundle format")
	require.Zero(t, timeStamp)

	// test invalid format with wrong order
	timeStamp, err = getTimeStampPartFromFileName("repoName.bundle.20221102153359")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid syntax")
	require.Zero(t, timeStamp)
}

func TestCreateHost(t *testing.T) {
	bbHost, err := NewBitBucketHost(NewBitBucketHostInput{})
	require.NoError(t, err)
	require.Equal(t, bitbucketAPIURL, bbHost.getAPIURL())

	ghHost, err := NewGitHubHost(NewGitHubHostInput{
		APIURL: githubAPIURL,
	})
	require.NoError(t, err)
	require.Equal(t, githubAPIURL, ghHost.getAPIURL())

	glHost, err := NewGitlabHost(NewGitlabHostInput{
		APIURL: gitlabAPIURL,
	})
	require.NoError(t, err)
	require.Equal(t, gitlabAPIURL, glHost.getAPIURL())
}

func TestGetLatestBundlePath(t *testing.T) {
	// invalid directory
	bundlePath, err := getLatestBundlePath("invalid-directory")
	require.Empty(t, bundlePath)
	require.Contains(t, err.Error(), "backup path read failed")

	// empty directory
	dir, err := os.MkdirTemp(os.TempDir(), "soba-*")
	require.NoError(t, err)
	bundlePath, err = getLatestBundlePath(dir)
	require.Empty(t, bundlePath)
	require.Contains(t, err.Error(), "no bundle files found in path")

	// directory with two bundles
	bundlePath, err = getLatestBundlePath("testfiles/example-bundles")
	require.NoError(t, err)
	require.Equal(t, "testfiles/example-bundles/example.20221102202522.bundle", bundlePath)
}

func TestPruneBackups(t *testing.T) {
	backupDir := filepath.Join(os.TempDir(), "tmp_githosts-utils")
	defer func() {
		if err := deleteBackupsDir(backupDir); err != nil {

			return
		}
	}()

	dfDir := path.Join(backupDir, "github.com", "go-soba", "repo0")
	assert.NoError(t, os.MkdirAll(dfDir, 0o755), fmt.Sprintf("failed to create dummy files dir: %s", dfDir))

	dummyFiles := []string{"repo0.20200401111111.bundle", "repo0.20200201010111.bundle", "repo0.20200501010111.bundle", "repo0.20200401011111.bundle", "repo0.20200601011111.bundle"}
	var err error
	for _, df := range dummyFiles {
		dfPath := path.Join(dfDir, df)
		_, err = os.OpenFile(dfPath, os.O_RDONLY|os.O_CREATE, 0o666)
		assert.NoError(t, err, fmt.Sprintf("failed to open file: %s", dfPath))
	}
	assert.NoError(t, pruneBackups(dfDir, 2))
	files, err := os.ReadDir(dfDir)
	assert.NoError(t, err)
	var found int
	notExpectedPostPrune := []string{"repo0.20200401111111.bundle", "repo0.20200201010111.bundle", "repo0.20200401011111.bundle"}
	expectedPostPrune := []string{"repo0.20200501010111.bundle", "repo0.20200601011111.bundle"}

	for _, f := range files {
		assert.NotContains(t, notExpectedPostPrune, f.Name())
		assert.Contains(t, expectedPostPrune, f.Name())
		found++
	}
	if found != 2 {
		t.Errorf("three backup files were expected")
	}
}

func TestPruneBackupsWithNonBundleFiles(t *testing.T) {
	backupDir := filepath.Join(os.TempDir(), "tmp_githosts-utils")
	defer func() {
		if err := deleteBackupsDir(backupDir); err != nil {

			return
		}
	}()

	dfDir := path.Join(backupDir, "github.com", "go-soba", "repo0")
	assert.NoError(t, os.MkdirAll(dfDir, 0o755), fmt.Sprintf("failed to create dummy files dir: %s", dfDir))

	dummyFiles := []string{"repo0.20200401111111.bundle", "repo0.20200201010111.bundle", "repo0.20200501010111.bundle", "repo0.20200401011111.bundle", "repo0.20200601011111.bundle", "repo0.20200601011111.bundle.lock"}
	var err error
	for _, df := range dummyFiles {
		dfPath := path.Join(dfDir, df)
		_, err = os.OpenFile(dfPath, os.O_RDONLY|os.O_CREATE, 0o666)
		assert.NoError(t, err, fmt.Sprintf("failed to open file: %s", dfPath))
	}

	assert.NoError(t, pruneBackups(dfDir, 2))
}

func TestTimeStampFromBundleName(t *testing.T) {
	timestamp, err := timeStampFromBundleName("reponame.20200401111111.bundle")
	assert.NoError(t, err)
	expected, err := time.Parse(timeStampFormat, "20200401111111")
	assert.NoError(t, err)
	assert.Equal(t, expected, timestamp)
}

func TestTimeStampFromBundleNameWithPeriods(t *testing.T) {
	timestamp, err := timeStampFromBundleName("repo.name.20200401111111.bundle")
	assert.NoError(t, err)
	expected, err := time.Parse(timeStampFormat, "20200401111111")
	assert.NoError(t, err)
	assert.Equal(t, expected, timestamp)
}

func TestTimeStampFromBundleNameReturnsErrorWithInvalidTimestamp(t *testing.T) {
	_, err := timeStampFromBundleName("reponame.2020.0401111111.bundle")
	assert.Error(t, err)
	assert.Equal(t, "bundle 'reponame.2020.0401111111.bundle' has an invalid timestamp", err.Error())
}

func TestGenerateMapFromRefsCmdOutput(t *testing.T) {
	// use a mixture of spaces and tabs to separate the sha from the ref
	// include tag ref with leading space
	// include invalid line with only a single entry
	// ensure pseudo ref HEAD is stripped
	input := `
	74e5977463007b3cb29ef11d776afa620e4e8698	    HEAD
	2b59eaba487acaa8a16467222520377cc09b5bac    	refs/heads/another-example
	74e5977463007b3cb29ef11d776afa620e4e8698 refs/heads/example
	2b59eaba487acaa8a16467222520377cc09b5bac												refs/tags/ dev_25#1^{}
	74e5977463007b3cb29ef11d776afa620e4e8698			refs/heads/master
	invalid
	`
	refs, err := generateMapFromRefsCmdOutput([]byte(input))
	require.NoError(t, err)
	require.Equal(t, "2b59eaba487acaa8a16467222520377cc09b5bac", refs["refs/tags/ dev_25#1^{}"])
	require.Equal(t, "2b59eaba487acaa8a16467222520377cc09b5bac", refs["refs/heads/another-example"])
	require.Equal(t, "74e5977463007b3cb29ef11d776afa620e4e8698", refs["refs/heads/example"])
	require.Equal(t, "74e5977463007b3cb29ef11d776afa620e4e8698", refs["refs/heads/master"])
}
