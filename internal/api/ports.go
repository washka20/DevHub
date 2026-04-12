package api

import (
	"context"
	"io"

	"devhub/internal/docker"
	"devhub/internal/git"
	"devhub/internal/gitlab"
)

// GitService defines the interface for git operations used by handlers.
type GitService interface {
	Status(dir string) (*git.GitStatus, error)
	BranchesDetailed(dir string) ([]git.BranchInfo, error)
	Log(dir string, limit int, offset int) ([]git.Commit, error)
	LogTopology(dir string) ([]git.TopologyNode, error)
	LogMetadata(dir string, limit int, offset int, branch string) ([]git.CommitMeta, error)
	Diff(dir string) (string, error)
	DiffFile(dir string, file string) (string, error)
	StageFiles(dir string, files []string) error
	UnstageFiles(dir string, files []string) error
	CommitChanges(dir string, message string, files []string) error
	Checkout(dir string, branch string) error
	Pull(dir string) (string, error)
	Push(dir string) (string, error)
	GenerateCommitMessage(dir string) (string, error)
	CommitDetail(dir string, hash string) (*git.CommitDetailInfo, error)
	CommitDiff(dir string, hash string, file string) (string, error)
	StashList(dir string) ([]git.StashEntry, error)
	StashPush(dir string, message string) error
	StashApply(dir string, index int) error
	StashPop(dir string, index int) error
	StashDrop(dir string, index int) error
	StashDiff(dir string, index int) (string, error)
	RemoteURL(dir string) (string, error)
	Blame(dir, filePath string) ([]git.BlameEntry, error)
	CherryPick(dir, hash string) error
}

// DockerService defines the interface for docker operations used by handlers.
type DockerService interface {
	Containers(composeFile string) ([]docker.Container, error)
	Action(composeFile string, containerName string, action string) error
	StreamLogs(ctx context.Context, composeFile, containerName string, tail int) (<-chan string, <-chan error)
	ComposeUp(composeFile string) (string, error)
	ComposeUpBuild(composeFile string) (string, error)
	ComposeDown(composeFile string) (string, error)
	Stats(composeFile string) ([]docker.ContainerStats, error)
	Inspect(composeFile, serviceName string) (*docker.ContainerInspect, error)
}

// GitLabClient defines the interface for GitLab API operations used by handlers.
type GitLabClient interface {
	ProjectByRemote(remoteURL string) (*gitlab.Project, error)
	Issues(projectID int, state string) ([]gitlab.Issue, error)
	MergeRequests(projectID int, state string) ([]gitlab.MergeRequest, error)
	Pipelines(projectID int) ([]gitlab.Pipeline, error)
	IssueDetail(projectID, iid int) (*gitlab.Issue, error)
	IssueNotes(projectID, iid int) ([]gitlab.Note, error)
	AddIssueNote(projectID, iid int, body string) (*gitlab.Note, error)
	UpdateIssue(projectID, iid int, req gitlab.UpdateIssueRequest) (*gitlab.Issue, error)
	CreateIssue(projectID int, req gitlab.CreateIssueRequest) (*gitlab.Issue, error)
	MRNotes(projectID, iid int) ([]gitlab.Note, error)
	AddMRNote(projectID, iid int, body string) (*gitlab.Note, error)
	CreateMR(projectID int, req gitlab.CreateMRRequest) (*gitlab.MergeRequest, error)
	ProjectMembers(projectID int) ([]gitlab.ProjectMember, error)
	MyIssues(state string) ([]gitlab.Issue, error)
	MyMergeRequests(state string) ([]gitlab.MergeRequest, error)
	CurrentUser() (*gitlab.Author, error)
	AllLabels() ([]gitlab.Label, error)
	AllMilestones() ([]gitlab.Milestone, error)
	FetchRaw(targetURL string) (io.ReadCloser, string, int64, error)
	MyMergeRequestsToReview(state string) ([]gitlab.MergeRequest, error)
	MRApprovals(projectID, iid int) (*gitlab.MRApproval, error)
	ApproveMR(projectID, iid int) error
	UnapproveMR(projectID, iid int) error
	PipelineJobs(projectID, pipelineID int) ([]gitlab.Job, error)
	JobTrace(projectID, jobID int) (string, error)
	RetryJob(projectID, jobID int) (*gitlab.Job, error)
	CancelJob(projectID, jobID int) (*gitlab.Job, error)
	MRDiscussions(projectID, iid int) ([]gitlab.Discussion, error)
	ResolveMRDiscussion(projectID, iid int, discussionID string, resolved bool) error
	ReplyToDiscussion(projectID, iid int, discussionID, body string) (*gitlab.DiscussionNote, error)
	MyTodos() ([]gitlab.Todo, error)
	MarkTodoDone(todoID int) error
	MarkAllTodosDone() error
	BaseURL() string
}
