package macos

import (
	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners/ios"
	"github.com/bitrise-core/bitrise-init/utility"
)

//------------------
// ScannerInterface
//------------------

// Scanner ...
type Scanner struct {
	searchDir         string
	configDescriptors []ios.ConfigDescriptor
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (Scanner) Name() string {
	return string(utility.XcodeProjectTypeMacOS)
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	scanner.searchDir = searchDir

	detected, err := ios.Detect(utility.XcodeProjectTypeMacOS, searchDir)
	if err != nil {
		return false, err
	}

	return detected, nil
}

// ExcludedScannerNames ...
func (Scanner) ExcludedScannerNames() []string {
	return []string{}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	options, configDescriptors, warnings, err := ios.GenerateOptions(utility.XcodeProjectTypeMacOS, scanner.searchDir)
	if err != nil {
		return models.OptionModel{}, warnings, err
	}

	scanner.configDescriptors = configDescriptors

	return options, warnings, nil
}

// DefaultOptions ...
func (Scanner) DefaultOptions() models.OptionModel {
	return ios.GenerateDefaultOptions(utility.XcodeProjectTypeMacOS)
}

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	return ios.GenerateConfig(utility.XcodeProjectTypeMacOS, scanner.configDescriptors, true)
}

// DefaultConfigs ...
func (Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	return ios.GenerateDefaultConfig(utility.XcodeProjectTypeMacOS, true)
}
