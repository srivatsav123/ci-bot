package merger

import (
	"github.com/Huawei-PaaS/ci-bot/handlers/dbm"
	"github.com/golang/glog"
	"github.com/google/go-github/github"
)

const (
	TableName = "issue"
)

type Merge struct {
	ID      string `orm:"column(id);null; type(text); pk"`
	Login   string `orm:"column(login);null; type(text)"`
	Name    string `orm:"column(name);null; type(text) "`
	Email   string `orm:"column(email);null; type(text) "`
	Count   int64  `orm:"column(count);null type(int)"`
	Repo    string `orm:"column(repo);null; type(text) "`
	PullReq int    `orm:"column(pullreq);null type(int)"`
}

type Issue struct {
	Userid int64  `orm:"column(id);null type(int); pk"`
	Name   string `orm:"column(name);null; type(text) "`
	Email  string `orm:"column(email);null; type(text) "`
	Count  int64  `orm:"column(count);null type(int)"`
}

//var bool merge

func init() {

	dbm.RegisterModel("issue", &Issue{})

	dbm.RegisterModel("issue", &Merge{})
	dbm.InitDBManager()

}

func HandleIssue(event github.IssuesEvent) error {

	authorEvent := new(Issue)

	authorEvent.Name = *event.Issue.User.Login
	authorEvent.Email = *event.Issue.User.Login
	authorEvent.Userid = *event.Issue.User.ID

	authors, err := QuerydbIssue(authorEvent)
	if err != nil {
		glog.Errorf("error is %v", err)
	}
	err = validateDbIssue(authors, authorEvent)
	if err != nil {
		glog.Errorf("error is %v", err)
	}

	return nil
}

func QuerydbIssue(author *Issue) (*Issue, error) {
	Prauthor := &Issue{}
	_, err := dbm.DBAccess.QueryTable(TableName).Filter("id", author.Userid).All(Prauthor)
	if err != nil {
		return nil, err
	}
	return Prauthor, nil

}

func validateDbIssue(author *Issue, prEvent *Issue) error {
	var count int64

	if author.Userid == prEvent.Userid {

		//update db
		count = (author.Count) + 1
		var val = make(map[string]interface{})
		val["count"] = count
		num, err := dbm.DBAccess.QueryTable(TableName).Filter("id", author.Userid).Update(val)
		if err != nil {
			glog.Errorf("failed to query table")
			return err
		}
		glog.Info("Update affected Num: %d, %s", num, err)
		//return err

	} else {
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
