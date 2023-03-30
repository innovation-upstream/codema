package fs

import "os"

func IsDir(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		// This means the file doesn't exist
		return false, nil
	}

	isDir := fileInfo.IsDir()

	return isDir, nil
}

func GetLegacyTemplatePath(basePath, templatePath string) string {
	tmplPath := basePath + templatePath
	return tmplPath
}

func GetTemplatePath(basePath, templateDir, templateVersion string) string {
	tmplPath := basePath + templateDir + "/" + templateVersion + ".template"
	return tmplPath
}
