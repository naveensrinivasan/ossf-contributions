package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type response struct {
	Title string `csv:"Title"`
	URL   string `csv:"URL"`
}

type PR struct {
	Title string
	URL   string
}

type PullRequestCreated struct {
	Data struct {
		Search struct {
			IssueCount int `json:"issueCount"`
			Edges      []struct {
				Node struct {
					Title string `json:"title"`
					URL   string `json:"url"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"search"`
	} `json:"data"`
}

type PullRequestReviewed struct {
	Data struct {
		Search struct {
			IssueCount int `json:"issueCount"`
			PageInfo   struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Edges []struct {
				Node struct {
					Title string `json:"title"`
					URL   string `json:"url"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"search"`
	} `json:"data"`
}

func main() {
	query := `
{
  search(
    query: "is:public  reviewed-by:naveensrinivasan created:2022-03-01..2022-03-30 user:ossf"
    type: ISSUE
    first: 100
  ) {
    issueCount
    pageInfo {
      hasNextPage
      endCursor
    }
    edges {
      node {
        ... on PullRequest {
          title
          url
        }
      }
    }
  }
}
`
	prs := `
{
  search(query: "is:pr is:public archived:false author:naveensrinivasan user:ossf created:2022-03-01..2022-03-30", type: ISSUE, first: 100) {
    issueCount
    edges {
      node {
        ... on PullRequest {
          number
          title
          url
        }
      }
    }
  }
}
`
	clientsFile, err := os.OpenFile("april.md", os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer clientsFile.Close()

	r := getPullRequestReviewed(query, os.Getenv("GITHUB_TOKEN"))
	// write to file mark down header pull requests reviewed by naveen for Feb
	fmt.Fprintf(clientsFile, "## Pull Requests Reviewed by Naveen Srinivasan for Jan\n")
	// include a counter
	fmt.Fprintf(clientsFile, "| # | Title | URL |\n")
	fmt.Fprintf(clientsFile, "| --- | --- | --- |\n")
	for i, edge := range r.Data.Search.Edges {
		fmt.Fprintf(clientsFile, "| %d | %s | %s |\n", i+1, edge.Node.Title, edge.Node.URL)
	}
	fmt.Fprintf(clientsFile, "\n")

	// write to file mark down header pull requests created by naveen for Feb
	fmt.Fprintf(clientsFile, "## Pull Requests Created by Naveen Srinivasan for Jan\n")
	fmt.Fprintf(clientsFile, "| # | Title | URL |\n")
	fmt.Fprintf(clientsFile, "| --- | --- | --- |\n")
	prsCreated := getPullRequestCreated(prs, os.Getenv("GITHUB_TOKEN"))
	for i, edge := range prsCreated.Data.Search.Edges {
		fmt.Fprintf(clientsFile, "| %d | %s | %s |\n", i+1, edge.Node.Title, edge.Node.URL)
	}
	fmt.Fprintf(clientsFile, "\n")

	issues := getIssuesCreated("user:ossf created:2022-03-01..2022-03-30 author:naveensrinivasan is:issue", os.Getenv("GITHUB_TOKEN"))
	fmt.Fprintf(clientsFile, "## Issues Created by Naveen Srinivasan for Jan\n")
	fmt.Fprintf(clientsFile, "| # | Title | URL |\n")
	fmt.Fprintf(clientsFile, "| --- | --- | --- |\n")
	for i, issue := range issues {
		fmt.Fprintf(clientsFile, "| %d | %s | %s |\n", i+1, issue.Title, issue.URL)
	}
	fmt.Fprintf(clientsFile, "\n")
}

func getPullRequestReviewed(q, token string) PullRequestReviewed {
	resp := getHttpResponse(q, token)
	x := &PullRequestReviewed{}
	if err := json.NewDecoder(resp.Body).Decode(x); err != nil {
		panic(err)
	}
	return *x
}

func getPullRequestCreated(q, token string) PullRequestCreated {
	resp := getHttpResponse(q, token)
	x := &PullRequestCreated{}
	if err := json.NewDecoder(resp.Body).Decode(x); err != nil {
		panic(err)
	}
	return *x
}

func getHttpResponse(q, token string) *http.Response {
	type Payload struct {
		Query string `json:"query"`
	}

	data := Payload{
		Query: q,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", "https://api.github.com/graphql", body)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	return resp
}

// use github api to search for issues created by naveen srinivasan in feb 2021 in the org ossf
func getIssuesCreated(q, token string) []PR {
	// use the go github api to search for issues created by naveen srinivasan in feb 2021 in the org ossf
	//
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)

	opt := &github.SearchOptions{
		TextMatch: true,
	}
	issues, _, err := githubClient.Search.Issues(context.Background(), q, opt)
	if err != nil {
		fmt.Println(err)
	}
	prs := []PR{}
	for _, issue := range issues.Issues {
		prs = append(prs, PR{
			Title: issue.GetTitle(),
			URL:   issue.GetHTMLURL(),
		})
	}
	return prs
}
