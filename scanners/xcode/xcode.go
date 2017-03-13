package xcode

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"path/filepath"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

const defaultConfigNameFormat = "default-%s-config"

const (
	projectPathKey    = "project_path"
	projectPathTitle  = "Project (or Workspace) path"
	projectPathEnvKey = "BITRISE_PROJECT_PATH"

	schemeKey    = "scheme"
	schemeTitle  = "Scheme name"
	schemeEnvKey = "BITRISE_SCHEME"

	carthageCommandKey   = "carthage_command"
	carthageCommandTitle = "Carthage command to run"
)

// ProjectType ...
type ProjectType string

const (
	// ProjectTypeiOS ...
	ProjectTypeiOS ProjectType = "ios"
	// ProjectTypemacOS ...
	ProjectTypemacOS ProjectType = "macos"
)

// ConfigDescriptor ...
type ConfigDescriptor struct {
	HasPodfile           bool
	CarthageCommand      string
	HasTest              bool
	MissingSharedSchemes bool
}

func (descriptor ConfigDescriptor) String() string {
	name := "-"
	if descriptor.HasPodfile {
		name = name + "pod-"
	}
	if descriptor.CarthageCommand != "" {
		name = name + "carthage-"
	}
	if descriptor.HasTest {
		name = name + "test-"
	}
	if descriptor.MissingSharedSchemes {
		name = name + "missing-shared-schemes-"
	}
	return name + "config"
}

// Scanner ...
type Scanner struct {
	searchDir         string
	fileList          []string
	projectFiles      []string
	configDescriptors []ConfigDescriptor
	projectType       ProjectType
}

// NewScanner ...
func NewScanner(projectType ProjectType) *Scanner {
	scanner := new(Scanner)
	scanner.projectType = projectType
	return scanner
}

// Name ...
func (scanner Scanner) Name() string {
	return string(scanner.projectType)
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	scanner.searchDir = searchDir

	fileList, err := utility.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", searchDir, err)
	}
	scanner.fileList = fileList

	// Search for xcodeproj
	log.Infoft("Searching for Xcode project files")

	xcodeprojectFiles, err := utility.FilterPaths(fileList, utility.AllowXcodeProjExtFilter)
	if err != nil {
		return false, err
	}

	log.Printft("%d Xcode project files found", len(xcodeprojectFiles))
	for _, xcodeprojectFile := range xcodeprojectFiles {
		log.Printft("- %s", xcodeprojectFile)
	}

	if len(xcodeprojectFiles) == 0 {
		log.Printft("platform not detected")
		return false, nil
	}

	log.Infoft("Filter relevant Xcode project files")

	filters := []utility.FilterFunc{
		utility.AllowIsDirectoryFilter,
		utility.ForbidEmbeddedWorkspaceRegexpFilter,
		utility.ForbidGitDirComponentFilter,
		utility.ForbidPodsDirComponentFilter,
		utility.ForbidCarthageDirComponentFilter,
		utility.ForbidFramworkComponentWithExtensionFilter,
	}

	switch scanner.projectType {
	case ProjectTypeiOS:
		filters = append(filters, utility.AllowIphoneosSDKFilter)
		break
	case ProjectTypemacOS:
		filters = append(filters, utility.AllowMacosxSDKFilter)
		break
	}

	xcodeprojectFiles, err = utility.FilterPaths(xcodeprojectFiles, filters...)
	if err != nil {
		return false, err
	}

	log.Printft("%d Xcode %s project files found", len(xcodeprojectFiles), scanner.Name())
	for _, xcodeprojectFile := range xcodeprojectFiles {
		log.Printft("- %s", xcodeprojectFile)
	}

	if len(xcodeprojectFiles) == 0 {
		log.Printft("platform not detected")
		return false, nil
	}

	scanner.projectFiles = xcodeprojectFiles

	log.Doneft("Platform detected")

	return true, nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	warnings := models.Warnings{}

	projectFiles := scanner.projectFiles

	filters := []utility.FilterFunc{
		utility.AllowIsDirectoryFilter,
		utility.ForbidEmbeddedWorkspaceRegexpFilter,
		utility.ForbidGitDirComponentFilter,
		utility.ForbidPodsDirComponentFilter,
		utility.ForbidCarthageDirComponentFilter,
		utility.ForbidFramworkComponentWithExtensionFilter,
		utility.AllowXCWorkspaceExtFilter,
	}

	switch scanner.projectType {
	case ProjectTypeiOS:
		filters = append(filters, utility.AllowIphoneosSDKFilter)
		break
	case ProjectTypemacOS:
		filters = append(filters, utility.AllowMacosxSDKFilter)
		break
	}

	workspaceFiles, err := utility.FilterPaths(scanner.fileList, filters...)
	if err != nil {
		return models.OptionModel{}, models.Warnings{}, err
	}

	standaloneProjects, workspaces, err := utility.CreateStandaloneProjectsAndWorkspaces(projectFiles, workspaceFiles)
	if err != nil {
		return models.OptionModel{}, models.Warnings{}, err
	}

	//
	// Create cocoapods workspace-project mapping
	log.Infoft("Searching for Podfiles")

	podfiles, err := utility.FilterPaths(scanner.fileList,
		utility.AllowPodfileBaseFilter,
		utility.ForbidGitDirComponentFilter,
		utility.ForbidPodsDirComponentFilter,
		utility.ForbidCarthageDirComponentFilter,
		utility.ForbidFramworkComponentWithExtensionFilter)
	if err != nil {
		return models.OptionModel{}, models.Warnings{}, err
	}

	log.Printft("%d Podfiles detected", len(podfiles))
	for _, file := range podfiles {
		log.Printft("- %s", file)
	}

	for _, podfile := range podfiles {
		workspaceProjectMap, err := utility.GetWorkspaceProjectMap(podfile, projectFiles)
		if err != nil {
			return models.OptionModel{}, models.Warnings{}, err
		}

		standaloneProjects, workspaces, err = utility.MergePodWorkspaceProjectMap(workspaceProjectMap, standaloneProjects, workspaces)
		if err != nil {
			return models.OptionModel{}, models.Warnings{}, err
		}
	}
	// ---

	//
	// Carthage
	log.Infof("Searching for Cartfile")

	cartfiles, err := utility.FilterPaths(scanner.fileList,
		utility.AllowCartfileBaseFilter,
		utility.ForbidGitDirComponentFilter,
		utility.ForbidPodsDirComponentFilter,
		utility.ForbidCarthageDirComponentFilter,
		utility.ForbidFramworkComponentWithExtensionFilter)
	if err != nil {
		return models.OptionModel{}, models.Warnings{}, err
	}

	log.Printf("%d Cartfiles detected", len(cartfiles))
	for _, file := range cartfiles {
		log.Printft("- %s", file)
	}
	// ----

	//
	// Analyze projects and workspaces
	isXcshareddataGitignored := false
	defaultGitignorePth := filepath.Join(scanner.searchDir, ".gitignore")

	if exist, err := pathutil.IsPathExists(defaultGitignorePth); err != nil {
		log.Warnf("Failed to check if .gitignore file exists at: %s, error: %s", defaultGitignorePth, err)
	} else if exist {
		isGitignored, err := utility.FileContains(defaultGitignorePth, "xcshareddata")
		if err != nil {
			log.Warnf("Failed to check if xcshareddata gitignored, error: %s", err)
		} else {
			isXcshareddataGitignored = isGitignored
		}
	}

	for _, project := range standaloneProjects {
		log.Infoft("Inspecting standalone project file: %s", project.Pth)

		log.Printft("%d shared schemes detected", len(project.SharedSchemes))
		for _, scheme := range project.SharedSchemes {
			log.Printft("- %s", scheme.Name)
		}

		if len(project.SharedSchemes) == 0 {
			log.Printft("")
			log.Errorft("No shared schemes found, adding recreate-user-schemes step...")
			log.Errorft("The newly generated schemes may differ from the ones in your project.")
			if isXcshareddataGitignored {
				log.Errorft("Your gitignore file (%s) contains 'xcshareddata', maybe shared schemes are gitignored?", defaultGitignorePth)
				log.Errorft("If not, make sure to share your schemes, to have the expected behaviour.")
			} else {
				log.Errorft("Make sure to share your schemes, to have the expected behaviour.")
			}
			log.Printft("")

			message := `No shared schemes found for project: ` + project.Pth + `.`
			if isXcshareddataGitignored {
				message += `
Your gitignore file (` + defaultGitignorePth + `) contains 'xcshareddata', maybe shared schemes are gitignored?`
			}
			message += `
Automatically generated schemes may differ from the ones in your project.
Make sure to <a href="http://devcenter.bitrise.io/ios/frequent-ios-issues/#xcode-scheme-not-found">share your schemes</a> for the expected behaviour.`

			warnings = append(warnings, message)

			log.Warnft("%d user schemes will be generated", len(project.Targets))
			for _, target := range project.Targets {
				log.Warnft("- %s", target.Name)
			}
		}
	}

	for _, workspace := range workspaces {
		log.Infoft("Inspecting workspace file: %s", workspace.Pth)

		sharedSchemes := workspace.GetSharedSchemes()
		log.Printft("%d shared schemes detected", len(sharedSchemes))
		for _, scheme := range sharedSchemes {
			log.Printft("- %s", scheme.Name)
		}

		if len(sharedSchemes) == 0 {
			log.Printft("")
			log.Errorft("No shared schemes found, adding recreate-user-schemes step...")
			log.Errorft("The newly generated schemes may differ from the ones in your project.")
			if isXcshareddataGitignored {
				log.Errorft("Your gitignore file (%s) contains 'xcshareddata', maybe shared schemes are gitignored?", defaultGitignorePth)
				log.Errorft("If not, make sure to share your schemes, to have the expected behaviour.")
			} else {
				log.Errorft("Make sure to share your schemes, to have the expected behaviour.")
			}
			log.Printft("")

			message := `No shared schemes found for project: ` + workspace.Pth + `.`
			if isXcshareddataGitignored {
				message += `
Your gitignore file (` + defaultGitignorePth + `) (%s) contains 'xcshareddata', maybe shared schemes are gitignored?`
			}
			message += `
Automatically generated schemes may differ from the ones in your project.
Make sure to <a href="http://devcenter.bitrise.io/ios/frequent-ios-issues/#xcode-scheme-not-found">share your schemes</a> for the expected behaviour.`

			warnings = append(warnings, message)

			targets := workspace.GetTargets()
			log.Warnft("%d user schemes will be generated", len(targets))
			for _, target := range targets {
				log.Warnft("- %s", target.Name)
			}
		}
	}
	// -----

	//
	// Create config descriptors
	configDescriptors := []ConfigDescriptor{}
	projectPathOption := models.NewOptionModel(projectPathTitle, projectPathEnvKey)

	// Add Standalon Project options
	for _, project := range standaloneProjects {
		schemeOption := models.NewOptionModel(schemeTitle, schemeEnvKey)

		carthageCommand := ""
		if utility.HasCartfileInDirectoryOf(project.Pth) {
			if utility.HasCartfileResolvedInDirectoryOf(project.Pth) {
				carthageCommand = "bootstrap"
			} else {
				dir := filepath.Dir(project.Pth)
				cartfilePth := filepath.Join(dir, "Cartfile")

				warnings = append(warnings, fmt.Sprintf(`Cartfile found at (%s), but no Cartfile.resolved exists in the same directory.
It is <a href="https://github.com/Carthage/Carthage/blob/master/Documentation/Artifacts.md#cartfileresolved">strongly recommended to commit this file to your repository</a>`, cartfilePth))

				carthageCommand = "update"
			}
		}

		if len(project.SharedSchemes) == 0 {
			for _, target := range project.Targets {
				configDescriptor := ConfigDescriptor{
					HasPodfile:           false,
					CarthageCommand:      carthageCommand,
					HasTest:              target.HasXCTest,
					MissingSharedSchemes: true,
				}
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewEmptyOptionModel()
				configOption.Config = scanner.Name() + configDescriptor.String()

				schemeOption.ValueMap[target.Name] = configOption
			}
		} else {
			for _, scheme := range project.SharedSchemes {
				configDescriptor := ConfigDescriptor{
					HasPodfile:           false,
					CarthageCommand:      carthageCommand,
					HasTest:              scheme.HasXCTest,
					MissingSharedSchemes: false,
				}
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewEmptyOptionModel()
				configOption.Config = scanner.Name() + configDescriptor.String()

				schemeOption.ValueMap[scheme.Name] = configOption
			}
		}

		projectPathOption.ValueMap[project.Pth] = schemeOption
	}

	// Add Workspace options
	for _, workspace := range workspaces {
		schemeOption := models.NewOptionModel(schemeTitle, schemeEnvKey)

		carthageCommand := ""
		if utility.HasCartfileInDirectoryOf(workspace.Pth) {
			if utility.HasCartfileResolvedInDirectoryOf(workspace.Pth) {
				carthageCommand = "bootstrap"
			} else {
				dir := filepath.Dir(workspace.Pth)
				cartfilePth := filepath.Join(dir, "Cartfile")

				warnings = append(warnings, fmt.Sprintf(`Cartfile found at (%s), but no Cartfile.resolved exists in the same directory.
It is <a href="https://github.com/Carthage/Carthage/blob/master/Documentation/Artifacts.md#cartfileresolved">strongly recommended to commit this file to your repository</a>`, cartfilePth))

				carthageCommand = "update"
			}
		}

		sharedSchemes := workspace.GetSharedSchemes()
		if len(sharedSchemes) == 0 {
			targets := workspace.GetTargets()
			for _, target := range targets {
				configDescriptor := ConfigDescriptor{
					HasPodfile:           workspace.IsPodWorkspace,
					CarthageCommand:      carthageCommand,
					HasTest:              target.HasXCTest,
					MissingSharedSchemes: true,
				}
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewEmptyOptionModel()
				configOption.Config = scanner.Name() + configDescriptor.String()

				schemeOption.ValueMap[target.Name] = configOption
			}
		} else {
			for _, scheme := range sharedSchemes {
				configDescriptor := ConfigDescriptor{
					HasPodfile:           workspace.IsPodWorkspace,
					CarthageCommand:      carthageCommand,
					HasTest:              scheme.HasXCTest,
					MissingSharedSchemes: false,
				}
				configDescriptors = append(configDescriptors, configDescriptor)

				configOption := models.NewEmptyOptionModel()
				configOption.Config = scanner.Name() + configDescriptor.String()

				schemeOption.ValueMap[scheme.Name] = configOption
			}
		}

		projectPathOption.ValueMap[workspace.Pth] = schemeOption
	}
	// -----

	if len(configDescriptors) == 0 {
		log.Errorft("No valid %s config found", scanner.Name())
		return models.OptionModel{}, warnings, fmt.Errorf("No valid %s config found", scanner.Name())
	}

	scanner.configDescriptors = configDescriptors

	return projectPathOption, warnings, nil
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionModel {
	configOption := models.NewEmptyOptionModel()
	configOption.Config = fmt.Sprintf(defaultConfigNameFormat, scanner.Name())

	projectPathOption := models.NewOptionModel(projectPathTitle, projectPathEnvKey)
	schemeOption := models.NewOptionModel(schemeTitle, schemeEnvKey)

	schemeOption.ValueMap["_"] = configOption
	projectPathOption.ValueMap["_"] = schemeOption

	return projectPathOption
}

func generateConfig(scanner *Scanner, hasPodfile, hasTest, missingSharedSchemes bool, carthageCommand string) bitriseModels.BitriseDataModel {
	//
	// Prepare steps
	prepareSteps := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	prepareSteps = append(prepareSteps, steps.ActivateSSHKeyStepListItem())

	// GitClone
	prepareSteps = append(prepareSteps, steps.GitCloneStepListItem())

	// Script
	prepareSteps = append(prepareSteps, steps.ScriptSteplistItem(steps.ScriptDefaultTitle))

	// CertificateAndProfileInstaller
	prepareSteps = append(prepareSteps, steps.CertificateAndProfileInstallerStepListItem())

	// CocoapodsInstall
	if hasPodfile {
		prepareSteps = append(prepareSteps, steps.CocoapodsInstallStepListItem())
	}

	// Carthage
	if carthageCommand != "" {
		prepareSteps = append(prepareSteps, steps.CarthageStepListItem([]envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{carthageCommandKey: carthageCommand},
		}))
	}

	// RecreateUserSchemes
	if missingSharedSchemes {
		prepareSteps = append(prepareSteps, steps.RecreateUserSchemesStepListItem([]envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		}))
	}
	// ----------

	envItemModels := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
	}

	//
	// CI steps
	ciSteps := append([]bitriseModels.StepListItemModel{}, prepareSteps...)

	// XcodeTest
	if hasTest {
		switch scanner.projectType {
		case ProjectTypeiOS:
			ciSteps = append(ciSteps, steps.XcodeTestStepListItem(envItemModels))
			break
		case ProjectTypemacOS:
			ciSteps = append(ciSteps, steps.XcodeTestMacStepListItem(envItemModels))
			break
		}
	}

	// DeployToBitriseIo
	ciSteps = append(ciSteps, steps.DeployToBitriseIoStepListItem())
	// ----------

	//
	// Deploy steps
	deploySteps := append([]bitriseModels.StepListItemModel{}, prepareSteps...)

	// XcodeTest
	if hasTest {
		switch scanner.projectType {
		case ProjectTypeiOS:
			deploySteps = append(deploySteps, steps.XcodeTestStepListItem(envItemModels))
			break
		case ProjectTypemacOS:
			deploySteps = append(deploySteps, steps.XcodeTestMacStepListItem(envItemModels))
			break
		}
	}

	// XcodeArchive
	switch scanner.projectType {
	case ProjectTypeiOS:
		deploySteps = append(deploySteps, steps.XcodeArchiveStepListItem(envItemModels))
		break
	case ProjectTypemacOS:
		deploySteps = append(deploySteps, steps.XcodeArchiveMacStepListItem(envItemModels))
		break
	}

	// DeployToBitriseIo
	deploySteps = append(deploySteps, steps.DeployToBitriseIoStepListItem())
	// ----------

	return models.BitriseDataWithCIAndCDWorkflow([]envmanModels.EnvironmentItemModel{}, ciSteps, deploySteps)
}

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	descriptors := []ConfigDescriptor{}
	descritorNameMap := map[string]bool{}

	for _, descriptor := range scanner.configDescriptors {
		_, exist := descritorNameMap[scanner.Name()+descriptor.String()]
		if !exist {
			descriptors = append(descriptors, descriptor)
		}
	}

	bitriseDataMap := models.BitriseConfigMap{}
	for _, descriptor := range descriptors {
		configName := scanner.Name() + descriptor.String()
		bitriseData := generateConfig(scanner, descriptor.HasPodfile, descriptor.HasTest, descriptor.MissingSharedSchemes, descriptor.CarthageCommand)
		data, err := yaml.Marshal(bitriseData)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}
		bitriseDataMap[configName] = string(data)
	}

	return bitriseDataMap, nil
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	//
	// Prepare steps
	prepareSteps := []bitriseModels.StepListItemModel{}

	// ActivateSSHKey
	prepareSteps = append(prepareSteps, steps.ActivateSSHKeyStepListItem())

	// GitClone
	prepareSteps = append(prepareSteps, steps.GitCloneStepListItem())

	// Script
	prepareSteps = append(prepareSteps, steps.ScriptSteplistItem(steps.ScriptDefaultTitle))

	// CertificateAndProfileInstaller
	prepareSteps = append(prepareSteps, steps.CertificateAndProfileInstallerStepListItem())

	// CocoapodsInstall
	prepareSteps = append(prepareSteps, steps.CocoapodsInstallStepListItem())

	// RecreateUserSchemes
	prepareSteps = append(prepareSteps, steps.RecreateUserSchemesStepListItem([]envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
	}))
	// ----------

	//
	// CI steps
	ciSteps := append([]bitriseModels.StepListItemModel{}, prepareSteps...)

	envItemModels := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{projectPathKey: "$" + projectPathEnvKey},
		envmanModels.EnvironmentItemModel{schemeKey: "$" + schemeEnvKey},
	}

	// XcodeTest
	switch scanner.projectType {
	case ProjectTypeiOS:
		ciSteps = append(ciSteps, steps.XcodeTestStepListItem(envItemModels))
		break
	case ProjectTypemacOS:
		ciSteps = append(ciSteps, steps.XcodeTestMacStepListItem(envItemModels))
		break
	}

	// DeployToBitriseIo
	ciSteps = append(ciSteps, steps.DeployToBitriseIoStepListItem())
	// ----------

	//
	// Deploy steps
	deploySteps := append([]bitriseModels.StepListItemModel{}, prepareSteps...)

	// XcodeTest
	switch scanner.projectType {
	case ProjectTypeiOS:
		deploySteps = append(deploySteps, steps.XcodeTestStepListItem(envItemModels))
		break
	case ProjectTypemacOS:
		deploySteps = append(deploySteps, steps.XcodeTestMacStepListItem(envItemModels))
		break
	}

	// XcodeArchive
	switch scanner.projectType {
	case ProjectTypeiOS:
		deploySteps = append(deploySteps, steps.XcodeArchiveStepListItem(envItemModels))
		break
	case ProjectTypemacOS:
		deploySteps = append(deploySteps, steps.XcodeArchiveMacStepListItem(envItemModels))
		break
	}

	// DeployToBitriseIo
	deploySteps = append(deploySteps, steps.DeployToBitriseIoStepListItem())
	// ----------

	config := models.BitriseDataWithCIAndCDWorkflow([]envmanModels.EnvironmentItemModel{}, ciSteps, deploySteps)
	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configName := fmt.Sprintf(defaultConfigNameFormat, scanner.Name())
	bitriseDataMap := models.BitriseConfigMap{}
	bitriseDataMap[configName] = string(data)

	return bitriseDataMap, nil
}