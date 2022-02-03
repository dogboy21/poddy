package models

type RepositoryProvider interface {
	GetSelfUser() (User, error)
	GetProject(slug string) (Project, error)
	DoesProjectBranchExist(slug, branchName string) (bool, error)
	GetProjectFile(slug, ref, path string) ([]byte, error)
}

type User interface {
	GetUsername() string
	GetDisplayName() string
	GetEmail() string
	GetAvatarUrl() string
	GetIsAdmin() bool
}

type Project interface {
	GetFullName() string
	GetHttpCloneUrl() string
	GetDefaultBranch() string
}
