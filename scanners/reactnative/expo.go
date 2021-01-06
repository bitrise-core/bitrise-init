package reactnative

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/bitrise-init/steps"
	"github.com/bitrise-io/bitrise-init/utility"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	expoConfigNamePublishNo         = "react-native-expo-config"
	expoConfigNamePublishYes        = "react-native-expo-config-publish"
	expoDefaultConfigNamePublishNo  = "default-" + expoConfigNamePublishNo
	expoDefaultConfigNamePublishYes = "default-" + expoConfigNamePublishYes
)

const (
	bareIOSProjectPathInputTitle   = "The iOS project path generated by the 'expo eject' process"
	bareIOSprojectPathInputSummary = "The relative location of the Xcode workspace, after running 'expo eject'. For example: './ios/myproject.xcworkspace'. Needed to eject to the bare workflow, managed workflow is not supported (https://docs.expo.io/bare/customizing/)."
)

const (
	iosBundleIDInputTitle          = "iOS bundle identifier"
	iosBundleIDInputSummary        = "Did not found the key expo/ios/bundleIdentifier in 'app.json'. You can add it now, or commit to the repository later. Needed to eject to the bare workflow, managed workflow is not supported (https://docs.expo.io/bare/customizing/)."
	iosBundleIDInputSummaryDefault = "You can specify the iOS bundle identifier in case the key expo/ios/bundleIdentifier is not set in 'app.json'. Needed to eject to the bare workflow, managed workflow is not supported (https://docs.expo.io/bare/customizing/)."
	iosBundleIDEnvKey              = "EXPO_BARE_IOS_BUNLDE_ID"
)

const (
	androidPackageInputTitle          = "Android package name"
	androidPackageInputSummary        = "Did not found the key expo/android/package in 'app.json'. You can add it now, or commit to the repository later. Needed to eject to the bare workflow, managed workflow is not supported (https://docs.expo.io/bare/customizing/)."
	androidPackageInputSummaryDefault = "You can specify the Android package name in case the key key expo/android/package is not set in 'app.json'. Needed to eject to the bare workflow, managed workflow is not supported (https://docs.expo.io/bare/customizing/)."
	androidPackageEnvKey              = "EXPO_BARE_ANDROID_PACKAGE"
)

const (
	iosDevelopmentTeamInputTitle   = "iOS Development team"
	iosDevelopmentTeamInputSummary = "The Apple Development Team that the iOS version of the app belongs to."
)

const (
	projectRootDirInputTitle   = "Project root directory"
	projectRootDirInputSummary = "The directory of the 'app.json' or 'package.json' file of your React Native project."
)

const (
	expoShouldPublishInputTitle   = "Publish Expo project?"
	expoShouldPublishInputSummary = "Will ask for Expo password and username in the next step."
)

const (
	expoUserNameInputTitle   = "Expo username"
	expoUserNameInputSummary = "Your Expo account username: required to publish using Expo CLI."
)

const (
	expoPasswordInputTitle   = "Expo password"
	expoPasswordInputSummary = "Your Expo account password: required to publish using Expo CLI."
)

const (
	schemeInputTitle   = "The iOS scheme name generated by the 'expo eject' process"
	schemeInputSummary = "An Xcode scheme defines a collection of targets to build, a configuration to use when building, and a collection of tests to execute. You can change the scheme at any time."
)

const (
	expoBareAddIdentiferScriptTitle = "Set bundleIdentifier, packageName for Expo Eject"
	expoAppJSONName                 = "app.json"
)

func expoBareAddIdentifiersScript(appJSONPath, androidEnvKey, iosEnvKey string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
set -ex

appJson="%s"
tmp="/tmp/app.json"
jq '.expo.android |= if has("package") or env.`+androidEnvKey+` == "" or env.`+androidEnvKey+` == null then . else .package = env.`+androidEnvKey+` end |
.expo.ios |= if has("bundleIdentifier") or env.`+iosEnvKey+` == "" or env.`+iosEnvKey+` == null then . else .bundleIdentifier = env.`+iosEnvKey+` end' <${appJson} >${tmp}
[[ $?==0 ]] && mv -f ${tmp} ${appJson}`, appJSONPath)
}

func appJSONError(appJSONPth, reason, explanation string) error {
	return fmt.Errorf("app.json file (%s) %s\n%s", appJSONPth, reason, explanation)
}

// expoOptions implements ScannerInterface.Options function for Expo based React Native projects.
func (scanner *Scanner) expoOptions() (models.OptionNode, models.Warnings, error) {
	warnings := models.Warnings{}
	log.TPrintf("Project name: %v", scanner.expoSettings.name)

	if scanner.expoSettings == nil {
		return models.OptionNode{}, warnings, errors.New("can not generate expo Options, expoSettings is nil")
	}

	var iosNode *models.OptionNode
	var exportMethodOption *models.OptionNode
	if scanner.expoSettings.isIOS { // ios options
		schemeOption := models.NewOption(ios.SchemeInputTitle, ios.SchemeInputSummary, ios.SchemeInputEnvKey, models.TypeSelector)

		// predict the ejected project name
		projectName := strings.ToLower(regexp.MustCompile(`(?i:[^a-z0-9])`).ReplaceAllString(scanner.expoSettings.name, ""))
		iosProjectInputType := models.TypeOptionalSelector
		if projectName == "" {
			iosProjectInputType = models.TypeUserInput
		}
		projectPathOption := models.NewOption(ios.ProjectPathInputTitle, bareIOSprojectPathInputSummary, ios.ProjectPathInputEnvKey, iosProjectInputType)
		if projectName != "" {
			projectPathOption.AddOption(filepath.Join("./", "ios", projectName+".xcworkspace"), schemeOption)
		} else {
			projectPathOption.AddOption("./ios/< PROJECT NAME >.xcworkspace", schemeOption)
		}

		if scanner.expoSettings.bundleIdentifierIOS == "" { // bundle ID Option
			iosNode = models.NewOption(iosBundleIDInputTitle, iosBundleIDInputSummary, iosBundleIDEnvKey, models.TypeUserInput)
			iosNode.AddOption("", projectPathOption)
		} else {
			iosNode = projectPathOption
		}

		developmentTeamOption := models.NewOption(iosDevelopmentTeamInputTitle, iosDevelopmentTeamInputSummary, "BITRISE_IOS_DEVELOPMENT_TEAM", models.TypeUserInput)
		schemeOption.AddOption(projectName, developmentTeamOption)

		exportMethodOption = models.NewOption(ios.IosExportMethodInputTitle, ios.IosExportMethodInputSummary, ios.ExportMethodInputEnvKey, models.TypeSelector)
		developmentTeamOption.AddOption("", exportMethodOption)
	}

	var androidNode *models.OptionNode
	var buildVariantOption *models.OptionNode
	if scanner.expoSettings.isAndroid { // android options
		packageJSONDir := filepath.Dir(scanner.packageJSONPth)
		relPackageJSONDir, err := utility.RelPath(scanner.searchDir, packageJSONDir)
		if err != nil {
			return models.OptionNode{}, warnings, fmt.Errorf("Failed to get relative package.json dir path, error: %s", err)
		}
		if relPackageJSONDir == "." {
			// package.json placed in the search dir, no need to change-dir in the workflows
			relPackageJSONDir = ""
		}

		var projectSettingNode *models.OptionNode
		var moduleOption *models.OptionNode
		if relPackageJSONDir == "" {
			projectSettingNode = models.NewOption(android.ProjectLocationInputTitle, android.ProjectLocationInputSummary, android.ProjectLocationInputEnvKey, models.TypeSelector)

			moduleOption = models.NewOption(android.ModuleInputTitle, android.ModuleInputSummary, android.ModuleInputEnvKey, models.TypeUserInput)
			projectSettingNode.AddOption("./android", moduleOption)
		} else {
			projectSettingNode = models.NewOption(projectRootDirInputTitle, projectRootDirInputSummary, "WORKDIR", models.TypeSelector)

			projectLocationOption := models.NewOption(android.ProjectLocationInputTitle, android.ProjectLocationInputSummary, android.ProjectLocationInputEnvKey, models.TypeSelector)
			projectSettingNode.AddOption(relPackageJSONDir, projectLocationOption)

			moduleOption = models.NewOption(android.ModuleInputTitle, android.ModuleInputSummary, android.ModuleInputEnvKey, models.TypeUserInput)
			projectLocationOption.AddOption(filepath.Join(relPackageJSONDir, "android"), moduleOption)
		}

		if scanner.expoSettings.packageNameAndroid == "" {
			androidNode = models.NewOption(androidPackageInputTitle, androidPackageInputSummary, androidPackageEnvKey, models.TypeUserInput)
			androidNode.AddOption("", projectSettingNode)
		} else {
			androidNode = projectSettingNode
		}

		buildVariantOption = models.NewOption(android.VariantInputTitle, android.VariantInputSummary, android.VariantInputEnvKey, models.TypeOptionalUserInput)
		moduleOption.AddOption("app", buildVariantOption)
	}

	allPlatformOptionsPublishNo := iosNode
	if iosNode != nil {
		if androidNode != nil {
			for _, exportMethod := range ios.IosExportMethods {
				exportMethodOption.AddOption(exportMethod, androidNode)
			}
		}
	} else {
		allPlatformOptionsPublishNo = androidNode
	}

	allPlatformOptionsPublishYes := allPlatformOptionsPublishNo.Copy()
	type configPair struct {
		node, config *models.OptionNode
	}
	for _, s := range []configPair{
		{node: allPlatformOptionsPublishNo, config: models.NewConfigOption(expoConfigNamePublishNo, nil)},
		{node: allPlatformOptionsPublishYes, config: models.NewConfigOption(expoConfigNamePublishYes, nil)},
	} {
		for _, lastOption := range s.node.LastChilds() {
			lastOption.ChildOptionMap = map[string]*models.OptionNode{}
			if androidNode != nil {
				// Android buildVariantOption is last
				lastOption.AddConfig("Release", s.config)
				continue
			}

			// iOS exportMethodOption is last
			for _, exportMethod := range ios.IosExportMethods {
				lastOption.AddConfig(exportMethod, s.config)
			}
		}
	}

	// expo options
	usernameOption := models.NewOption(expoUserNameInputTitle, expoUserNameInputSummary, "EXPO_USERNAME", models.TypeUserInput)
	passwordOption := models.NewOption(expoPasswordInputTitle, expoPasswordInputSummary, "EXPO_PASSWORD", models.TypeUserInput)
	usernameOption.AddOption("", passwordOption)
	rootNode := models.NewOption(expoShouldPublishInputTitle, expoShouldPublishInputSummary, "", models.TypeSelector)

	if scanner.hasTest { // If there are no tests, there is only a primary workflow, do not publish
		rootNode.AddOption("yes", usernameOption)
	}
	rootNode.AddOption("no", allPlatformOptionsPublishNo)
	passwordOption.AddOption("", allPlatformOptionsPublishYes)

	return *rootNode, warnings, nil
}

// expoConfigs implements ScannerInterface.Configs function for Expo based React Native projects.
func (scanner *Scanner) expoConfigs() (models.BitriseConfigMap, error) {
	configMap := models.BitriseConfigMap{}

	// determine workdir
	packageJSONDir := filepath.Dir(scanner.packageJSONPth)
	relPackageJSONDir, err := utility.RelPath(scanner.searchDir, packageJSONDir)
	if err != nil {
		return models.BitriseConfigMap{}, fmt.Errorf("Failed to get relative package.json dir path, error: %s", err)
	}
	if relPackageJSONDir == "." {
		// package.json placed in the search dir, no need to change-dir in the workflows
		relPackageJSONDir = ""
	}
	log.TPrintf("Working directory: %v", relPackageJSONDir)

	workdirEnvList := []envmanModels.EnvironmentItemModel{}
	if relPackageJSONDir != "" {
		workdirEnvList = append(workdirEnvList, envmanModels.EnvironmentItemModel{workDirInputKey: relPackageJSONDir})
	}

	if !scanner.hasTest {
		// if the project has no test script defined,
		// we can only provide deploy like workflow,
		// so that is going to be the primary workflow

		configBuilder := models.NewDefaultConfigBuilder()
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)

		if scanner.hasYarnLockFile {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		} else {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		}

		projectDir := relPackageJSONDir
		if relPackageJSONDir == "" {
			projectDir = "./"
		}

		if !scanner.expoSettings.isAllIdentifierPresent() {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.ScriptSteplistItem(expoBareAddIdentiferScriptTitle,
				envmanModels.EnvironmentItemModel{"content": expoBareAddIdentifiersScript(filepath.Join(projectDir, expoAppJSONName), androidPackageEnvKey, iosBundleIDEnvKey)},
			))
		}

		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.ExpoDetachStepListItem(
			envmanModels.EnvironmentItemModel{"project_path": projectDir},
		))

		// android build
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
			envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
		))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.AndroidBuildStepListItem(
			envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: "$" + android.ProjectLocationInputEnvKey},
			envmanModels.EnvironmentItemModel{android.ModuleInputKey: "$" + android.ModuleInputEnvKey},
			envmanModels.EnvironmentItemModel{android.VariantInputKey: "$" + android.VariantInputEnvKey},
		))

		// ios build
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.XcodeArchiveStepListItem(
			envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
			envmanModels.EnvironmentItemModel{ios.ExportMethodInputKey: "$" + ios.ExportMethodInputEnvKey},
			envmanModels.EnvironmentItemModel{"force_team_id": "$BITRISE_IOS_DEVELOPMENT_TEAM"},
		))

		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)
		configBuilder.SetWorkflowDescriptionTo(models.PrimaryWorkflowID, deployWorkflowDescription)

		bitriseDataModel, err := configBuilder.Generate(scannerName)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(bitriseDataModel)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		configMap[expoConfigNamePublishNo] = string(data)

		return configMap, nil
	}

	for _, config := range []struct {
		isPublish bool
		ID        string
	}{
		{isPublish: false, ID: expoConfigNamePublishNo},
		{isPublish: true, ID: expoConfigNamePublishYes},
	} {
		// primary workflow
		configBuilder := models.NewDefaultConfigBuilder()
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)
		if scanner.hasYarnLockFile {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "test"})...))
		} else {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "test"})...))
		}
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

		// deploy workflow
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(false)...)
		if scanner.hasYarnLockFile {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		} else {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		}

		projectDir := relPackageJSONDir
		if relPackageJSONDir == "" {
			projectDir = "./"
		}

		if !scanner.expoSettings.isAllIdentifierPresent() {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.ScriptSteplistItem(expoBareAddIdentiferScriptTitle,
				envmanModels.EnvironmentItemModel{"content": expoBareAddIdentifiersScript(filepath.Join(projectDir, expoAppJSONName), androidPackageEnvKey, iosBundleIDEnvKey)},
			))
		}

		if config.isPublish {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.ExpoDetachStepListItem(
				envmanModels.EnvironmentItemModel{"project_path": projectDir},
				envmanModels.EnvironmentItemModel{"user_name": "$EXPO_USERNAME"},
				envmanModels.EnvironmentItemModel{"password": "$EXPO_PASSWORD"},
				envmanModels.EnvironmentItemModel{"run_publish": "yes"},
			))
		} else {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.ExpoDetachStepListItem(
				envmanModels.EnvironmentItemModel{"project_path": projectDir},
			))
		}

		// android build
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
			envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
		))
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidBuildStepListItem(
			envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: "$" + android.ProjectLocationInputEnvKey},
			envmanModels.EnvironmentItemModel{android.ModuleInputKey: "$" + android.ModuleInputEnvKey},
			envmanModels.EnvironmentItemModel{android.VariantInputKey: "$" + android.VariantInputEnvKey},
		))

		// ios build
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(
			envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
			envmanModels.EnvironmentItemModel{ios.ExportMethodInputKey: "$" + ios.ExportMethodInputEnvKey},
			envmanModels.EnvironmentItemModel{"force_team_id": "$BITRISE_IOS_DEVELOPMENT_TEAM"},
		))

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)
		configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)

		bitriseDataModel, err := configBuilder.Generate(scannerName)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(bitriseDataModel)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		configMap[config.ID] = string(data)
	}

	return configMap, nil
}

// expoDefaultOptions implements ScannerInterface.DefaultOptions function for Expo based React Native projects.
func (Scanner) expoDefaultOptions() models.OptionNode {
	// ios options
	projectPathOptionPublishNo := models.NewOption(bareIOSProjectPathInputTitle, bareIOSprojectPathInputSummary, ios.ProjectPathInputEnvKey, models.TypeOptionalUserInput)

	bundleIDOption := models.NewOption(iosBundleIDInputTitle, iosBundleIDInputSummaryDefault, iosBundleIDEnvKey, models.TypeUserInput)
	projectPathOptionPublishNo.AddOption("./ios/< PROJECT NAME >.xcworkspace", bundleIDOption)

	schemeOption := models.NewOption(schemeInputTitle, schemeInputSummary, ios.SchemeInputEnvKey, models.TypeUserInput)
	bundleIDOption.AddOption("", schemeOption)

	exportMethodOption := models.NewOption(ios.IosExportMethodInputTitle, ios.IosExportMethodInputSummary, ios.ExportMethodInputEnvKey, models.TypeSelector)
	schemeOption.AddOption("", exportMethodOption)

	// android options
	androidPackageOption := models.NewOption(androidPackageInputTitle, androidPackageInputSummaryDefault, androidPackageEnvKey, models.TypeOptionalUserInput)
	for _, exportMethod := range ios.IosExportMethods {
		exportMethodOption.AddOption(exportMethod, androidPackageOption)
	}

	workDirOption := models.NewOption(projectRootDirInputTitle, projectRootDirInputSummary, "WORKDIR", models.TypeUserInput)
	androidPackageOption.AddOption("", workDirOption)

	projectLocationOption := models.NewOption(android.ProjectLocationInputTitle, android.ProjectLocationInputSummary, android.ProjectLocationInputEnvKey, models.TypeSelector)
	workDirOption.AddOption("", projectLocationOption)

	moduleOption := models.NewOption(android.ModuleInputTitle, android.ModuleInputSummary, android.ModuleInputEnvKey, models.TypeUserInput)
	projectLocationOption.AddOption("./android", moduleOption)

	buildVariantOption := models.NewOption(android.VariantInputTitle, android.VariantInputSummary, android.VariantInputEnvKey, models.TypeOptionalUserInput)
	moduleOption.AddOption("app", buildVariantOption)

	projectPathOptionPublishYes := projectPathOptionPublishNo.Copy()
	// Expo CLI options
	shouldPublishNode := models.NewOption(expoShouldPublishInputTitle, expoShouldPublishInputSummary, "", models.TypeSelector)

	userNameOption := models.NewOption(expoUserNameInputTitle, expoUserNameInputSummary, "EXPO_USERNAME", models.TypeUserInput)
	shouldPublishNode.AddOption("yes", userNameOption)

	passwordOption := models.NewOption(expoPasswordInputTitle, expoPasswordInputSummary, "EXPO_PASSWORD", models.TypeUserInput)
	userNameOption.AddOption("", passwordOption)

	passwordOption.AddOption("", projectPathOptionPublishYes)
	shouldPublishNode.AddOption("no", projectPathOptionPublishNo)

	for _, s := range []struct{ option, config *models.OptionNode }{
		{option: projectPathOptionPublishNo, config: models.NewConfigOption(expoDefaultConfigNamePublishNo, nil)},
		{option: projectPathOptionPublishYes, config: models.NewConfigOption(expoDefaultConfigNamePublishYes, nil)},
	} {
		for _, lastOption := range s.option.LastChilds() {
			lastOption.ChildOptionMap = map[string]*models.OptionNode{}
			// buildVariantOption is the last Option added
			lastOption.AddConfig("Release", s.config)
		}
	}

	return *shouldPublishNode
}

// expoDefaultConfigs implements ScannerInterface.DefaultConfigs function for Expo based React Native projects.
func (Scanner) expoDefaultConfigs() (models.BitriseConfigMap, error) {
	configMap := models.BitriseConfigMap{}

	for _, config := range []struct {
		isPublish bool
		ID        string
	}{
		{isPublish: false, ID: expoDefaultConfigNamePublishNo},
		{isPublish: true, ID: expoDefaultConfigNamePublishYes},
	} {
		// primary workflow
		configBuilder := models.NewDefaultConfigBuilder()

		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{workDirInputKey: "$WORKDIR"}, envmanModels.EnvironmentItemModel{"command": "install"}))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{workDirInputKey: "$WORKDIR"}, envmanModels.EnvironmentItemModel{"command": "test"}))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

		// deploy workflow
		configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(false)...)
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{workDirInputKey: "$WORKDIR"}, envmanModels.EnvironmentItemModel{"command": "install"}))

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.ScriptSteplistItem(expoBareAddIdentiferScriptTitle,
			envmanModels.EnvironmentItemModel{"content": expoBareAddIdentifiersScript(filepath.Join(".", expoAppJSONName), androidPackageEnvKey, iosBundleIDEnvKey)},
		))

		if config.isPublish {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.ExpoDetachStepListItem(
				envmanModels.EnvironmentItemModel{"project_path": "$WORKDIR"},
				envmanModels.EnvironmentItemModel{"user_name": "$EXPO_USERNAME"},
				envmanModels.EnvironmentItemModel{"password": "$EXPO_PASSWORD"},
				envmanModels.EnvironmentItemModel{"run_publish": "yes"},
			))
		} else {
			configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.ExpoDetachStepListItem(
				envmanModels.EnvironmentItemModel{"project_path": "$WORKDIR"},
			))
		}

		// android build
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
			envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
		))
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidBuildStepListItem(
			envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: "$" + android.ProjectLocationInputEnvKey},
			envmanModels.EnvironmentItemModel{android.ModuleInputKey: "$" + android.ModuleInputEnvKey},
			envmanModels.EnvironmentItemModel{android.VariantInputKey: "$" + android.VariantInputEnvKey},
		))

		// ios build
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(
			envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.ExportMethodInputKey: "$" + ios.ExportMethodInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
		))

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

		bitriseDataModel, err := configBuilder.Generate(scannerName)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(bitriseDataModel)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		configMap[config.ID] = string(data)
	}

	return configMap, nil
}
