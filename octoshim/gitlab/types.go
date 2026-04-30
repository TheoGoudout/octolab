package gitlab

type MR struct {
	ID           int    `json:"id"`
	IID          int    `json:"iid"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	State        string `json:"state"`
	Draft        bool   `json:"draft"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	SHA          string `json:"sha"`
	WebURL       string `json:"web_url"`
	MergeStatus  string `json:"merge_status"`
	Author       User   `json:"author"`
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type Note struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
	Author User  `json:"author"`
}

type CommitStatus struct {
	ID          int    `json:"id"`
	SHA         string `json:"sha"`
	Status      string `json:"status"`
	Name        string `json:"name"`
	Description string `json:"description"`
	TargetURL   string `json:"target_url"`
}

type Issue struct {
	ID    int    `json:"id"`
	IID   int    `json:"iid"`
	Title string `json:"title"`
	Body  string `json:"description"`
	WebURL string `json:"web_url"`
}

type Commit struct {
	ID             string `json:"id"`
	ShortID        string `json:"short_id"`
	Title          string `json:"title"`
	Message        string `json:"message"`
	AuthorName     string `json:"author_name"`
	AuthorEmail    string `json:"author_email"`
	CommittedDate  string `json:"committed_date"`
	WebURL         string `json:"web_url"`
}

type Label struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}
