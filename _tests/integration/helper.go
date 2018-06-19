package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"

	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func binPath() string {
	return os.Getenv("INTEGRATION_TEST_BINARY_PATH")
}

func extractFormat(str string, formats ...interface{}) (string, error) {
	for _, f := range formats {
		if format, ok := f.(string); ok {
			beforeCount := strings.Count(str, format)
			if beforeCount < 1 {
				return "", fmt.Errorf("format's original value not found")
			}
			str = strings.Replace(str, format, "%s", 1)

			afterCount := strings.Count(str, format)
			if beforeCount-1 != afterCount {
				return "", fmt.Errorf("failed to extract all formats")
			}
		}
	}
	return str, nil
}

func validateConfigExpectation(t *testing.T, ID, expected, actual string, versions ...interface{}) {
	if !assert.ObjectsAreEqual(expected, actual) {
		s, err := extractFormat(actual, versions...)
		require.NoError(t, err)
		fmt.Println("---------------------")
		fmt.Println("Expected config:")
		fmt.Println("---------------------")
		fmt.Println(expected)
		fmt.Println("---------------------")
		fmt.Println("---------------------")
		fmt.Println("Actual config:")
		fmt.Println("---------------------")
		fmt.Println(actual)
		fmt.Println("---------------------")
		fmt.Println("---------------------")
		fmt.Println("Actual config format:")
		fmt.Println("---------------------")
		fmt.Println(s)
		fmt.Println("---------------------")

		_, err = exec.LookPath("opendiff")
		if err == nil {
			tmpDir, err := pathutil.NormalizedOSTempDirPath("__diffs__")
			require.NoError(t, err)
			expPth := filepath.Join(tmpDir, ID+"-expected.yml")
			actPth := filepath.Join(tmpDir, ID+"-actual.yml")
			require.NoError(t, fileutil.WriteStringToFile(expPth, expected))
			require.NoError(t, fileutil.WriteStringToFile(actPth, actual))
			err = exec.Command("opendiff", expPth, actPth).Start()
			require.NoError(t, err)
			t.FailNow()
			return
		}
		log.Warnf("opendiff not installed, unable to open config diff")
		t.FailNow()
	}
}
