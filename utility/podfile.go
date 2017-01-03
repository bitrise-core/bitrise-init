package utility

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"encoding/json"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-tools/go-xcode/xcodeproj"
)

const podfileBase = "Podfile"

// AllowPodfileBaseFilter ...
var AllowPodfileBaseFilter = BaseFilter(podfileBase, true)

func getTargetDefinitionProjectMap(podfilePth string) (map[string]string, error) {
	gemfileContent := `source 'https://rubygems.org'
gem 'cocoapods-core'
gem 'json'
`
	// returns target - project map, if xcodeproj defined in the Podfile
	// return empty string if no xcodeproj defined in the Podfile
	rubyScriptContent := `require 'cocoapods-core'
require 'json'

begin
	podfile_path = ENV['PODFILE_PATH']
	podfile = Pod::Podfile.from_file(podfile_path)
	targets = podfile.target_definitions
	
	puts "#{{}.to_json}" unless targets

	target_project_map = {}
	targets.each do |name, target_definition|
		next unless target_definition.user_project_path
		target_project_map[name] = target_definition.user_project_path
	end

	puts "#{{ :data => target_project_map }.to_json}"
rescue => e
	puts "#{{ :error => e.to_s }.to_json}"
end
`

	envs := []string{fmt.Sprintf("PODFILE_PATH=%s", podfilePth)}
	podfileDir := filepath.Dir(podfilePth)

	out, err := runRubyScriptForOutput(rubyScriptContent, gemfileContent, podfileDir, envs)
	if err != nil {
		return map[string]string{}, err
	}

	fmt.Printf("%v\n", out)

	if out == "" {
		return map[string]string{}, nil
	}

	var targetDefinitionProjectMap map[string]string
	if err := json.Unmarshal([]byte(out), &targetDefinitionProjectMap); err != nil {
		return map[string]string{}, err
	}

	return targetDefinitionProjectMap, nil
}

func getUserDefinedProjectRelavtivePath(podfilePth string) (string, error) {
	targetProjectMap, err := getTargetDefinitionProjectMap(podfilePth)
	if err != nil {
		return "", err
	}

	for target, project := range targetProjectMap {
		if target == "Pods" {
			return project, nil
		}
	}
	return "", nil
}

func getUserDefinedWorkspaceRelativePath(podfilePth string) (string, error) {
	getWorkspacePathGemfileContent := `source 'https://rubygems.org'
gem 'cocoapods-core'
`

	// returns WORKSPACE_NAME.xcworkspace if user defined a workspace name
	// returns empty struct {}, if no user defined workspace name exists in Podfile
	getWorkspacePathRubyScriptContent := `require 'cocoapods-core'
podfile_path = ENV['PODFILE_PATH']
podfile = Pod::Podfile.from_file(podfile_path)
puts podfile.workspace_path
`

	envs := []string{fmt.Sprintf("PODFILE_PATH=%s", podfilePth)}
	podfileDir := filepath.Dir(podfilePth)

	workspaceBase, err := runRubyScriptForOutput(getWorkspacePathRubyScriptContent, getWorkspacePathGemfileContent, podfileDir, envs)
	if err != nil {
		return "", err
	}

	return workspaceBase, nil
}

// GetWorkspaceProjectMap ...
// If one project exists in the Podfile's directory, workspace name will be the project's name.
// If more then one project exists in the Podfile's directory, root 'xcodeproj/project' property have to be defined in the Podfile.
// Root 'xcodeproj/project' property will be mapped to the default cocoapods target (Pods).
// If workspace property defined in the Podfile, it will override the workspace name.
func GetWorkspaceProjectMap(podfilePth string, projects []string) (map[string]string, error) {
	// fix podfile quotation
	podfileContent, err := fileutil.ReadStringFromFile(podfilePth)
	if err != nil {
		return map[string]string{}, err
	}

	podfileContent = strings.Replace(podfileContent, `‘`, `'`, -1)
	podfileContent = strings.Replace(podfileContent, `’`, `'`, -1)
	podfileContent = strings.Replace(podfileContent, `“`, `"`, -1)
	podfileContent = strings.Replace(podfileContent, `”`, `"`, -1)

	if err := fileutil.WriteStringToFile(podfilePth, podfileContent); err != nil {
		return map[string]string{}, err
	}
	// ----

	podfileDir := filepath.Dir(podfilePth)

	projectRelPth, err := getUserDefinedProjectRelavtivePath(podfilePth)
	if err != nil {
		return map[string]string{}, err
	}

	if projectRelPth == "" {
		projects, err := FilterPaths(projects, InDirectoryFilter(podfileDir, true))
		if err != nil {
			return map[string]string{}, err
		}

		if len(projects) == 0 {
			return map[string]string{}, errors.New("failed to determin workspace - project mapping: no explicit project specified and no project found in the Podfile's directory")
		} else if len(projects) > 1 {
			return map[string]string{}, errors.New("failed to determin workspace - project mapping: no explicit project specified and more than one project found in the Podfile's directory")
		}

		projectRelPth = filepath.Base(projects[0])
	}
	projectPth := filepath.Join(podfileDir, projectRelPth)

	workspaceRelPth, err := getUserDefinedWorkspaceRelativePath(podfilePth)
	if err != nil {
		return map[string]string{}, err
	}

	if workspaceRelPth == "" {
		projectName := filepath.Base(strings.TrimSuffix(projectPth, ".xcodeproj"))
		workspaceRelPth = projectName + ".xcworkspace"
	}
	workspacePth := filepath.Join(podfileDir, workspaceRelPth)

	return map[string]string{
		workspacePth: projectPth,
	}, nil
}

// MergePodWorkspaceProjectMap ...
// Previously we separated standalone projects and workspaces.
// But pod workspace-project map may define workspace which is not in the repository, but will be created by `pod install`.
// This case the related project should be found in the standalone projects list.
// We will create this workspace model, add the related project and remove this project from standlone projects.
// If workspace is in the repository, both workspace and project should be find in the input lists.
func MergePodWorkspaceProjectMap(podWorkspaceProjectMap map[string]string, standaloneProjects []xcodeproj.ProjectModel, workspaces []xcodeproj.WorkspaceModel) ([]xcodeproj.ProjectModel, []xcodeproj.WorkspaceModel, error) {
	mergedStandaloneProjects := []xcodeproj.ProjectModel{}
	mergedWorkspaces := []xcodeproj.WorkspaceModel{}

	for podWorkspaceFile, podProjectFile := range podWorkspaceProjectMap {
		podWorkspace, found := FindWorkspaceInList(podWorkspaceFile, workspaces)
		if found {
			// Workspace found, this means workspace is in the repository.
			// This case the project is already attached to the workspace.
			podWorkspace.IsPodWorkspace = true

			_, found := FindProjectInList(podProjectFile, podWorkspace.Projects)
			if !found {
				return []xcodeproj.ProjectModel{}, []xcodeproj.WorkspaceModel{}, fmt.Errorf("pod workspace (%s) found, but assigned project (%s) project not", podWorkspaceFile, podProjectFile)
			}

			mergedStandaloneProjects = standaloneProjects
			mergedWorkspaces = workspaces
		} else {
			// Workspace not found, this means workspace is not in the repository,
			// but it will created by `pod install`.
			// This case the pod project was marked previously as standalone project.
			podWorkspace = xcodeproj.WorkspaceModel{
				Pth:            podWorkspaceFile,
				Name:           strings.TrimSuffix(filepath.Base(podWorkspaceFile), filepath.Ext(podWorkspaceFile)),
				IsPodWorkspace: true,
			}

			podProject, found := FindProjectInList(podProjectFile, standaloneProjects)
			if !found {
				return []xcodeproj.ProjectModel{}, []xcodeproj.WorkspaceModel{}, fmt.Errorf("pod workspace (%s) will be generated by (%s) project, but it does not found", podWorkspaceFile, podProjectFile)
			}

			podWorkspace.Projects = []xcodeproj.ProjectModel{podProject}

			mergedStandaloneProjects = RemoveProjectFromList(podProjectFile, standaloneProjects)
			mergedWorkspaces = append(workspaces, podWorkspace)
		}
	}

	return mergedStandaloneProjects, mergedWorkspaces, nil
}
