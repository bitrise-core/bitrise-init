package android

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-core/bitrise-init/utility"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/log"
)

// ScannerName ...
const ScannerName = "android"

const (
	configName        = "android-config"
	defaultConfigName = "default-android-config"
)

// Step Inputs
const (
	gradlewPathInputKey    = "gradlew_path"
	gradlewPathInputEnvKey = "GRADLEW_PATH"
	gradlewPathInputTitle  = "Gradlew file path"
)

const (
	pathInputKey          = "path"
	gradlewDirInputEnvKey = "GRADLEW_DIR_PATH"
	gradlewDirInputTitle  = "Directory of gradle wrapper"
)

const (
	gradleFileInputKey    = "gradle_file"
	gradleFileInputEnvKey = "GRADLE_BUILD_FILE_PATH"
	gradleFileInputTitle  = "Path to the gradle file to use"
)

const (
	gradleTaskInputKey    = "gradle_task"
	gradleTaskInputEnvKey = "GRADLE_TASK"
	gradleTaskInputTitle  = "Gradle task to run"
)

var defaultGradleTasks = []string{
	"assemble",
	"assembleDebug",
	"assembleRelease",
}

//------------------
// ScannerInterface
//------------------

// Scanner ...
type Scanner struct {
	FileList    []string
	GradleFiles []string
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (scanner Scanner) Name() string {
	return ScannerName
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	fileList, err := utility.ListPathInDirSortedByComponents(searchDir, true)
	if err != nil {
		return false, fmt.Errorf("failed to search for files in (%s), error: %s", searchDir, err)
	}
	scanner.FileList = fileList

	// Search for gradle file
	log.Infoft("Searching for build.gradle files")

	gradleFiles, err := utility.FilterRootBuildGradleFiles(fileList)
	if err != nil {
		return false, fmt.Errorf("failed to search for build.gradle files, error: %s", err)
	}
	scanner.GradleFiles = gradleFiles

	log.Printft("%d build.gradle files detected", len(gradleFiles))
	for _, file := range gradleFiles {
		log.Printft("- %s", file)
	}

	if len(gradleFiles) == 0 {
		log.Printft("platform not detected")
		return false, nil
	}

	log.Doneft("Platform detected")

	return true, nil
}

// ExcludedScannerNames ...
func (scanner *Scanner) ExcludedScannerNames() []string {
	return []string{}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	// Search for gradlew_path input
	log.Infoft("Searching for gradlew files")

	warnings := models.Warnings{}
	gradlewFiles, err := utility.FilterGradlewFiles(scanner.FileList)
	if err != nil {
		return models.OptionModel{}, warnings, fmt.Errorf("Failed to list gradlew files, error: %s", err)
	}

	log.Printft("%d gradlew files detected", len(gradlewFiles))
	for _, file := range gradlewFiles {
		log.Printft("- %s", file)
	}

	rootGradlewPath := ""
	gradlewFilesCount := len(gradlewFiles)
	switch {
	case gradlewFilesCount == 0:
		log.Errorft("No gradle wrapper (gradlew) found")
		return models.OptionModel{}, warnings, fmt.Errorf(`<b>No Gradle Wrapper (gradlew) found.</b> 
Using a Gradle Wrapper (gradlew) is required, as the wrapper is what makes sure
that the right Gradle version is installed and used for the build. More info/guide: <a>https://docs.gradle.org/current/userguide/gradle_wrapper.html</a>`)
	case gradlewFilesCount == 1:
		rootGradlewPath = gradlewFiles[0]
	case gradlewFilesCount > 1:
		rootGradlewPath = gradlewFiles[0]
		log.Warnft("Multiple gradlew file, detected:")
		for _, gradlewPth := range gradlewFiles {
			log.Warnft("- %s", gradlewPth)
		}
		log.Warnft("Using: %s", rootGradlewPath)
	}

	// Inspect Gradle files
	gradleFileOption := models.NewOption(gradleFileInputTitle, gradleFileInputEnvKey)

	for _, gradleFile := range scanner.GradleFiles {
		log.Infoft("Inspecting gradle file: %s", gradleFile)

		// generate-gradle-wrapper step will generate the wrapper
		gradleFileDir := filepath.Dir(gradleFile)
		rootGradlewPath = filepath.Join(gradleFileDir, "gradlew")
		// ---

		gradlewPthOption := models.NewOption(gradlewPathInputTitle, gradlewPathInputEnvKey)

		projectRootDirOption := models.NewOption(gradlewDirInputTitle, gradlewDirInputEnvKey)
		gradleFileOption.AddOption(gradleFile, projectRootDirOption)

		projectRootDirOption.AddOption(filepath.Join("$BITRISE_SOURCE_DIR", filepath.Dir(gradleFile)), gradlewPthOption)

		gradleTaskOption := models.NewOption(gradleTaskInputTitle, gradleTaskInputEnvKey)
		gradlewPthOption.AddOption(rootGradlewPath, gradleTaskOption)

		log.Printft("%d gradle tasks", len(defaultGradleTasks))

		for _, gradleTask := range defaultGradleTasks {
			log.Printft("- %s", gradleTask)

			configOption := models.NewConfigOption(configName)
			gradleTaskOption.AddConfig(gradleTask, configOption)
		}
	}

	return *gradleFileOption, warnings, nil
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionModel {
	gradleFileOption := models.NewOption(gradleFileInputTitle, gradleFileInputEnvKey)

	projectRootOption := models.NewOption(gradlewDirInputTitle, gradlewDirInputEnvKey)
	gradleFileOption.AddOption("_", projectRootOption)

	gradlewPthOption := models.NewOption(gradlewPathInputTitle, gradlewPathInputEnvKey)
	projectRootOption.AddOption("_", gradlewPthOption)

	gradleTaskOption := models.NewOption(gradleTaskInputTitle, gradleTaskInputEnvKey)
	gradlewPthOption.AddOption("_", gradleTaskOption)

	configOption := models.NewConfigOption(defaultConfigName)
	gradleTaskOption.AddConfig("_", configOption)

	return *gradleFileOption
}

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	configBuilder.AppendPreparStepList(steps.ChangeWorkDirStepListItem(envmanModels.EnvironmentItemModel{pathInputKey: "$" + gradlewDirInputEnvKey}))
	configBuilder.AppendPreparStepList(steps.InstallMissingAndroidToolsStepListItem())
	configBuilder.AppendMainStepList(steps.GradleRunnerStepListItem(
		envmanModels.EnvironmentItemModel{gradleFileInputKey: "$" + gradleFileInputEnvKey},
		envmanModels.EnvironmentItemModel{gradleTaskInputKey: "$" + gradleTaskInputEnvKey},
		envmanModels.EnvironmentItemModel{gradlewPathInputKey: "$" + gradlewPathInputEnvKey},
	))

	config, err := configBuilder.Generate(ScannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		configName: string(data),
	}, nil
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	configBuilder.AppendPreparStepList(steps.ChangeWorkDirStepListItem(envmanModels.EnvironmentItemModel{pathInputKey: "$" + gradlewDirInputEnvKey}))
	configBuilder.AppendPreparStepList(steps.InstallMissingAndroidToolsStepListItem())

	configBuilder.AppendMainStepList(steps.GradleRunnerStepListItem(
		envmanModels.EnvironmentItemModel{gradleFileInputKey: "$" + gradleFileInputEnvKey},
		envmanModels.EnvironmentItemModel{gradleTaskInputKey: "$" + gradleTaskInputEnvKey},
		envmanModels.EnvironmentItemModel{gradlewPathInputKey: "$" + gradlewPathInputEnvKey},
	))

	config, err := configBuilder.Generate(ScannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		defaultConfigName: string(data),
	}, nil
}
