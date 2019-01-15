package models

import (
	"errors"

	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
)

// WorkflowID ...
type WorkflowID string

const (
	// PrimaryWorkflowID ...
	PrimaryWorkflowID WorkflowID = "primary"
	// DeployWorkflowID ...
	DeployWorkflowID WorkflowID = "deploy"

	// FormatVersion ...
	FormatVersion = bitriseModels.Version

	defaultSteplibSource = "https://github.com/bitrise-io/bitrise-steplib.git"
)

// ConfigBuilderModel ...
type ConfigBuilderModel struct {
	workflowBuilderMap map[WorkflowID]*workflowBuilderModel
	triggerWorkflowID  WorkflowID
}

// NewDefaultConfigBuilder ...
func NewDefaultConfigBuilder() *ConfigBuilderModel {
	return &ConfigBuilderModel{
		workflowBuilderMap: map[WorkflowID]*workflowBuilderModel{
			PrimaryWorkflowID: newDefaultWorkflowBuilder(),
		},
		triggerWorkflowID: PrimaryWorkflowID,
	}
}

// NewCustomTriggerWorkflowIDConfigBuilder ...
func NewCustomTriggerWorkflowIDConfigBuilder(triggerWorkflowID WorkflowID) *ConfigBuilderModel {
	return &ConfigBuilderModel{
		workflowBuilderMap: map[WorkflowID]*workflowBuilderModel{
			triggerWorkflowID: newDefaultWorkflowBuilder(),
		},
		triggerWorkflowID: triggerWorkflowID,
	}
}

// AppendStepListItemsTo ...
func (builder *ConfigBuilderModel) AppendStepListItemsTo(workflow WorkflowID, items ...bitriseModels.StepListItemModel) {
	workflowBuilder := builder.workflowBuilderMap[workflow]
	if workflowBuilder == nil {
		workflowBuilder = newDefaultWorkflowBuilder()
		builder.workflowBuilderMap[workflow] = workflowBuilder
	}
	workflowBuilder.appendStepListItems(items...)
}

// SetWorkflowDescriptionTo ...
func (builder *ConfigBuilderModel) SetWorkflowDescriptionTo(workflow WorkflowID, description string) {
	workflowBuilder := builder.workflowBuilderMap[workflow]
	if workflowBuilder == nil {
		workflowBuilder = newDefaultWorkflowBuilder()
		builder.workflowBuilderMap[workflow] = workflowBuilder
	}
	workflowBuilder.Description = description
}

// Generate ...
func (builder *ConfigBuilderModel) Generate(projectType string, appEnvs ...envmanModels.EnvironmentItemModel) (bitriseModels.BitriseDataModel, error) {
	triggerWorkflowBuilder, ok := builder.workflowBuilderMap[builder.triggerWorkflowID]
	if !ok || triggerWorkflowBuilder == nil || len(triggerWorkflowBuilder.Steps) == 0 {
		return bitriseModels.BitriseDataModel{}, errors.New("primary trigger workflow not defined")
	}

	workflows := map[string]bitriseModels.WorkflowModel{}
	for workflowID, workflowBuilder := range builder.workflowBuilderMap {
		workflows[string(workflowID)] = workflowBuilder.generate()
	}

	triggerMap := []bitriseModels.TriggerMapItemModel{
		bitriseModels.TriggerMapItemModel{
			PushBranch: "*",
			WorkflowID: string(builder.triggerWorkflowID),
		},
		bitriseModels.TriggerMapItemModel{
			PullRequestSourceBranch: "*",
			WorkflowID:              string(builder.triggerWorkflowID),
		},
	}

	app := bitriseModels.AppModel{
		Environments: appEnvs,
	}

	return bitriseModels.BitriseDataModel{
		FormatVersion:        FormatVersion,
		DefaultStepLibSource: defaultSteplibSource,
		ProjectType:          projectType,
		TriggerMap:           triggerMap,
		Workflows:            workflows,
		App:                  app,
	}, nil
}
