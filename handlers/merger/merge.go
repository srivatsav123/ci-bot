package merger

import (
	"context"
	"github.com/Huawei-PaaS/ci-bot/handlers/dbm"
	"github.com/golang/glog"
	"github.com/google/go-github/github"
	"time"
)

const (
	TableNamemerge = "merge"
)

type CommitAuthor struct {
	Date  *time.Time `json:"date,omitempty"`
	Name  *string    `json:"name,omitempty"`
	Email *string    `json:"email,omitempty"`
	Login *string    `json:"username,omitempty"` // Renamed for go-github consistency.
}

type Commit struct {
	SHA       *string       `json:"sha,omitempty"`
	Author    *CommitAuthor `json:"author,omitempty"`
	Committer *CommitAuthor `json:"committer,omitempty"`
	Message   *string       `json:"message,omitempty"`
	Parents   []Commit      `json:"parents,omitempty"`
	HTMLURL   *string       `json:"html_url,omitempty"`
	URL       *string       `json:"url,omitempty"`
	NodeID    *string       `json:"node_id,omitempty"`

	// CommentCount is the number of GitHub comments on the commit. This
	// is only populated for requests that fetch GitHub data like
	// Pulls.ListCommits, Repositories.ListCommits, etc.
	CommentCount *int `json:"comment_count,omitempty"`
}

func HandleCheckPrmerged(prEvent github.PullRequestEvent, client *github.Client) error {
	authorEvent := new(Merge)
	if *(prEvent.PullRequest.Merged) == true {
		authorEvent.Userid = *(prEvent.PullRequest.User.ID)

		var local string
		local = *prEvent.PullRequest.Head.SHA
		commit, _, err := client.Git.GetCommit(context.Background(), "kubeedge", "kubeedge", local)
		if err != nil {
			glog.Infof("Git.GetCommit returned error: %v", err)
		}
		authorEvent.Email = *commit.Author.Email
		authorEvent.Name = *commit.Author.Name

		authors, err := Querydb(authorEvent)
		if err != nil {
			glog.Errorf("error is %v", err)
			return err
		}
		glog.Infof("after querying db is %v", authors)
		err = validateDb(authors, authorEvent)
		if err != nil {
			glog.Errorf("error is %v", err)
			return err
		}
	} else {
		glog.Info("pr is not merged")

	}

	return nil

}

func Querydb(author *Merge) (*Merge, error) {
	prauthor := &Merge{}
	_, err := dbm.DBAccess.QueryTable(TableNamemerge).Filter("id", author.Userid).All(prauthor)
	if err != nil {
		glog.Errorf("error is %v", err)
		return nil, err
	}
	glog.Infof("pr author is %v", prauthor)
	return prauthor, nil

}

func validateDb(author *Merge, prEvent *Merge) error {
	var count int64
	glog.Infof(" author.userid is %d", author.Userid)

	glog.Infof(" prEvent.Userid is %d", prEvent.Userid)
	if author.Userid == prEvent.Userid {

		//update db
		count = (author.Count) + 1
		var val = make(map[string]interface{})
		val["count"] = count
		num, err := dbm.DBAccess.QueryTable(TableNamemerge).Filter("id", author.Userid).Update(val)
		glog.Info("Update affected Num: %d, %s", num, err)

	} else {
		//insert
		(prEvent.Count) = 1
		num, err := dbm.DBAccess.Insert(prEvent)
		if err != nil {
			glog.Errorf("error is %v", err)
			return err
		}
		glog.Info("Insert affected Num: %d", num)

	}

	return nil
}
