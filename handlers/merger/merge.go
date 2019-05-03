package merger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Huawei-PaaS/ci-bot/handlers/dbm"
	"github.com/Huawei-PaaS/ci-bot/handlers/types"
	"github.com/golang/glog"
	"github.com/google/go-github/github"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	TableNamemerge = "merge"
)

var commitnames []string

type CommitAuthor struct {
	Date  *time.Time `json:"date,omitempty"`
	Name  *string    `json:"name,omitempty"`
	Email *string    `json:"email,omitempty"`
	Login *string    `json:"username,omitempty"` // Renamed for go-github consistency.
}

type Commit struct {
	SHA       *string       `json:"sha"`
	Author    *CommitAuthor `json:"author"`
	Committer *CommitAuthor `json:"committer"`
	Message   *string       `json:"message"`
	Parents   []Commit      `json:"parents"`
	HTMLURL   *string       `json:"html_url"`
	URL       *string       `json:"url"`
	NodeID    *string       `json:"node_id"`

	// CommentCount is the number of GitHub comments on the commit. This
	// is only populated for requests that fetch GitHub data like
	// Pulls.ListCommits, Repositories.ListCommits, etc.
	CommentCount *int `json:"comment_count"`
}

// contains returns whether slice contains elem.
func contains(slice []string, elem string) bool {
	for _, n := range slice {
		if elem == n {
			return true
		}
	}
	return false
}

//commiterLogin returns login of the commiter
func commiterLogin(commitResponse []types.Commits, sha string) (string, error) {
	for _, commitResp := range commitResponse {
		if reflect.DeepEqual(commitResp.Sha, sha) {
			return commitResp.Author.Login, nil
		}
	}
	err := errors.New("commitResponses does not contain the SHA.")
	return "", err
}

func commits(repoFullName string) []types.Commits {
	var commitBody []types.Commits
	client := &http.Client{}
	repoURL := "https://api.github.com/repos/" + repoFullName + "/commits"
	req, err := http.NewRequest("GET", repoURL, nil)
	if err != nil {
		glog.Infof("error: ", err)
	}
	// TODO: remove custom Accept headers when APIs fully launch.
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Infof("error: ", err)
	}
	err = json.Unmarshal(bodyBytes, &commitBody)
	if err != nil {
		glog.Infof("error: ", err)
	}
	defer resp.Body.Close()
	return commitBody
}

func HandleCheckPrmerged(prEvent github.PullRequestEvent, client *github.Client) error {
	commitnames = make([]string, 0)
	commitResponse := commits(*prEvent.Repo.FullName)
	if *(prEvent.PullRequest.Merged) == true {
		for i, _ := range commitResponse {
			var SHAs []string
			var mainSHA string
			var parentSHA string
			var continueFlag bool
			ShaFlag := false
			if reflect.DeepEqual(*prEvent.PullRequest.MergeCommitSHA, commitResponse[i].Sha) {
				mainSHA = commitResponse[i].Sha
				parentSHA = commitResponse[i].Parents[0].Sha
				ShaFlag = true
			}
			if ShaFlag == true {
				for _, commitResp := range commitResponse {
					continueFlag = false
					if reflect.DeepEqual(commitResp.Sha, mainSHA) {
						continue
					}
					if reflect.DeepEqual(commitResp.Sha, parentSHA) {
						break
					}
					for j, commitMsg := range strings.Split(commitResp.Commit.Message, "#") {
						if j == 0 && reflect.DeepEqual(commitMsg, "Merge pull request ") {
							continueFlag = true
						}
					}
					if continueFlag == true {
						continue
					}
					SHAs = append(SHAs, commitResp.Sha)
				}
			}
			for _, sha := range SHAs {
				authorEvent := new(Merge)
				fmt.Println("\n\ncommitResponse.sha: ", sha)
				commit, _, err := client.Git.GetCommit(context.Background(), *prEvent.Repo.Owner.Login, *prEvent.Repo.Name, sha)
				if err != nil {
					glog.Infof("Git.GetCommit returned error: ", err)
				}
				fmt.Println("Commit names before: ", commitnames)
				if contains(commitnames, *commit.Author.Name) == false {
					commitnames = append(commitnames, *commit.Author.Name)
				}
				fmt.Println("Commit names after: ", commitnames)
				fmt.Println("commit.Committer.Name:", *commit.Committer.Name)
				authorEvent.Login, err = commiterLogin(commitResponse, sha)
				if err != nil {
					glog.Errorf("error: ", err)
					return err
				}
				authorEvent.ID = *prEvent.Repo.FullName + "+" + strconv.Itoa(*prEvent.PullRequest.Number) + "+" + authorEvent.Login
				authorEvent.Repo = *prEvent.Repo.FullName
				authorEvent.Email = *commit.Author.Email
				authorEvent.Name = *commit.Author.Name
				authorEvent.PullReq = *prEvent.PullRequest.Number
				fmt.Println("authorEvent.ID: ", authorEvent.ID)
				fmt.Println("prEvent.PullRequest.URL: ", *prEvent.PullRequest.URL)
				fmt.Println("commit.Author.Name: ", *commit.Author.Name)
				authors, err := Querydb(authorEvent)
				fmt.Println("authors: ", *authors)
				if err != nil {
					glog.Errorf("error: ", err)
					return err
				}
				fmt.Println("after querying db: ", *authors)
				err = validateDb(authors, authorEvent)
				if err != nil {
					glog.Errorf("error: ", err)
					return err
				}
			}
		}
	} else {
		glog.Info("PR is not merged.")
	}
	return nil
}

func Querydb(author *Merge) (*Merge, error) {
	prauthor := &Merge{}
	fmt.Println("pr author: ", *prauthor)
	fmt.Println("inside Querydb: ", *author)
	_, err := dbm.DBAccess.QueryTable(TableNamemerge).Filter("id", author.ID).All(prauthor)
	if err != nil {
		glog.Errorf("error: ", err)
		return nil, err
	}
	fmt.Println("pr author: ", *prauthor)
	return prauthor, nil
}

func validateDb(author *Merge, prEvent *Merge) error {
	var count int64
	fmt.Println("author.Login: ", author.ID)
	fmt.Println("prEvent.Login: ", prEvent.ID)
	if reflect.DeepEqual(author.ID, prEvent.ID) {
		//update db
		count = (author.Count) + 1
		var val = make(map[string]interface{})
		val["count"] = count
		num, err := dbm.DBAccess.QueryTable(TableNamemerge).Filter("id", author.ID).Update(val)
		fmt.Println("update affected num, commiter: ", num, err)
	} else {
		//insert
		(prEvent.Count) = 1
		num, err := dbm.DBAccess.Insert(prEvent)
		if err != nil {
			glog.Errorf("error: ", err)
			return err
		}
		fmt.Println("insert affected Num: ", num)
	}
	return nil
}
