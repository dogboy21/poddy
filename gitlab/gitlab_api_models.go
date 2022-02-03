package gitlab

/* ================================================================================ */

type User struct {
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
	IsAdmin   bool   `json:"is_admin"`
}

func (u *User) GetUsername() string {
	return u.Username
}

func (u *User) GetDisplayName() string {
	return u.Name
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetAvatarUrl() string {
	return u.AvatarUrl
}

func (u *User) GetIsAdmin() bool {
	return u.IsAdmin
}

/* ================================================================================ */

type Project struct {
	PathWithNamespace string `json:"path_with_namespace"`
	HttpCloneUrl      string `json:"http_url_to_repo"`
	DefaultBranch     string `json:"default_branch"`
}

func (p *Project) GetFullName() string {
	return p.PathWithNamespace
}

func (p *Project) GetHttpCloneUrl() string {
	return p.HttpCloneUrl
}

func (p *Project) GetDefaultBranch() string {
	return p.DefaultBranch
}

/* ================================================================================ */

type RepositoryBranch struct {
	Name string `json:"name"`
}
