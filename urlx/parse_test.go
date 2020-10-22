package urlx

import (
	"testing"

	"github.com/ory/x/logrusx"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	type testData struct {
		urlStr       string
		expectedPath string
		expectedStr  string
	}
	var testURLs = []testData{
		{"File:///home/test/file1.txt", "/home/test/file1.txt", "file:///home/test/file1.txt"},
		{"fIle:/home/test/file2.txt", "/home/test/file2.txt", "file:///home/test/file2.txt"},
		{"fiLe:///../test/update/file3.txt", "/../test/update/file3.txt", "file:///../test/update/file3.txt"},
		{"filE://../test/update/file4.txt", "../test/update/file4.txt", "../test/update/file4.txt"},
		{"file://C:/users/test/file5.txt", "/C:/users/test/file5.txt", "file:///C:/users/test/file5.txt"},  // We expect a initial / in the path because this is a Windows absolute path
		{"file:///C:/users/test/file6.txt", "/C:/users/test/file6.txt", "file:///C:/users/test/file6.txt"}, // --//--
		{"file://file7.txt", "file7.txt", "file7.txt"},
		{"file://path/file8.txt", "path/file8.txt", "path/file8.txt"},
		{"file://C:\\Users\\RUNNER~1\\AppData\\Local\\Temp\\9ccf9f68-121c-451a-8a73-2aa360925b5a386398343/access-rules.json", "/C:/Users/RUNNER~1/AppData/Local/Temp/9ccf9f68-121c-451a-8a73-2aa360925b5a386398343/access-rules.json", "file:///C:/Users/RUNNER~1/AppData/Local/Temp/9ccf9f68-121c-451a-8a73-2aa360925b5a386398343/access-rules.json"},
		{"file:///C:\\Users\\RUNNER~1\\AppData\\Local\\Temp\\9ccf9f68-121c-451a-8a73-2aa360925b5a386398343/access-rules.json", "/C:/Users/RUNNER~1/AppData/Local/Temp/9ccf9f68-121c-451a-8a73-2aa360925b5a386398343/access-rules.json", "file:///C:/Users/RUNNER~1/AppData/Local/Temp/9ccf9f68-121c-451a-8a73-2aa360925b5a386398343/access-rules.json"},
		{"file://C:\\Users\\path with space\\file.txt", "/C:/Users/path with space/file.txt", "file:///C:/Users/path%20with%20space/file.txt"},
		{"file8b.txt", "file8b.txt", "file8b.txt"},
		{"../file9.txt", "../file9.txt", "../file9.txt"},
		{"./file9b.txt", "./file9b.txt", "./file9b.txt"},
		{"file://./file9c.txt", "./file9c.txt", "./file9c.txt"},
		{"file://./folder/.././file9d.txt", "./folder/.././file9d.txt", "./folder/.././file9d.txt"},
		{"..\\file10.txt", "../file10.txt", "../file10.txt"},
		{"C:\\file11.txt", "/C:/file11.txt", "file:///C:/file11.txt"},
		{"\\\\hostname\\share\\file12.txt", "/share/file12.txt", "file://hostname/share/file12.txt"},
		{"\\\\", "/", "file:///"},
		{"\\\\hostname", "/", "file://hostname/"},
		{"\\\\hostname\\", "/", "file://hostname/"},
		{"file:///home/test/file 13.txt", "/home/test/file 13.txt", "file:///home/test/file%2013.txt"},
		{"file:///home/test/file%2014.txt", "/home/test/file 14.txt", "file:///home/test/file%2014.txt"},
		{"http://server:80/test/file%2015.txt", "/test/file 15.txt", "http://server:80/test/file%2015.txt"},
	}

	for _, td := range testURLs {
		u, err := Parse(td.urlStr)
		assert.NoError(t, err)
		assert.Equal(t, td.expectedPath, u.Path, "expected path for %s", td.urlStr)
		assert.Equal(t, td.expectedStr, u.String(), "expected URL string for %s", td.urlStr)
	}

	assert.Panics(t, func() {
		ParseOrPanic("::")
	})
	assert.NotPanics(t, func() {
		ParseOrPanic(testURLs[0].urlStr)
	})
	exitCode := 0
	l := logrusx.New("", "", logrusx.WithExitFunc(func(c int) {
		exitCode = c
	}))
	ParseOrFatal(l, "::")
	assert.NotZero(t, exitCode, "ParseOrFatal should fail with a non zero exit code")
	ParseOrFatal(l, testURLs[0].urlStr)
	assert.NotZero(t, exitCode, "ParseOrFatal should not fail, zero exit code expected")

}
