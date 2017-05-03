package utility

import (
	"encoding/json"
	"encoding/xml"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
)

const configXMLBasePath = "config.xml"

// WidgetModel ...
type WidgetModel struct {
	XMLNSCDV string `xml:"xmlns cdv,attr"`
}

func parseConfigXMLContent(content string) (WidgetModel, error) {
	widget := WidgetModel{}
	if err := xml.Unmarshal([]byte(content), &widget); err != nil {
		return WidgetModel{}, err
	}
	return widget, nil
}

// ParseConfigXML ...
func ParseConfigXML(pth string) (WidgetModel, error) {
	content, err := fileutil.ReadStringFromFile(pth)
	if err != nil {
		return WidgetModel{}, err
	}
	return parseConfigXMLContent(content)
}

// FilterRootConfigXMLFile ...
func FilterRootConfigXMLFile(fileList []string) (string, error) {
	allowConfigXMLBaseFilter := BaseFilter(configXMLBasePath, true)
	configXMLs, err := FilterPaths(fileList, allowConfigXMLBaseFilter)
	if err != nil {
		return "", err
	}

	if len(configXMLs) == 0 {
		return "", nil
	}

	return configXMLs[0], nil
}

// PackagesModel ...
type PackagesModel struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func parsePackagesJSONContent(content string) (PackagesModel, error) {
	var packages PackagesModel
	if err := json.Unmarshal([]byte(content), &packages); err != nil {
		return PackagesModel{}, err
	}
	return packages, nil
}

// ParsePackagesJSON ...
func ParsePackagesJSON(packagesJSONPth string) (PackagesModel, error) {
	content, err := fileutil.ReadStringFromFile(packagesJSONPth)
	if err != nil {
		return PackagesModel{}, err
	}
	return parsePackagesJSONContent(content)
}

// RelCordovaWorkDir ...
func RelCordovaWorkDir(baseDir, cordovaConfigPth string) (string, error) {
	absBaseDir, err := pathutil.AbsPath(baseDir)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(absBaseDir, "/private/var") {
		absBaseDir = strings.TrimPrefix(absBaseDir, "/private")
	}

	absCordovaConfigPth, err := pathutil.AbsPath(cordovaConfigPth)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(absCordovaConfigPth, "/private/var") {
		absCordovaConfigPth = strings.TrimPrefix(absCordovaConfigPth, "/private")
	}

	absCordovaWorkDir := filepath.Dir(absCordovaConfigPth)
	if absBaseDir == absCordovaWorkDir {
		return "", nil
	}

	cordovaWorkdir, err := filepath.Rel(absBaseDir, absCordovaWorkDir)
	if err != nil {
		return "", err
	}

	return cordovaWorkdir, nil
}
