package main

import (
	"net/http"
	"octolab/octoshim/handlers"
	"strings"
)

type handlerFunc func(w http.ResponseWriter, r *http.Request, params map[string]string)

type route struct {
	method   string // "*" matches any
	segments []string
	handler  handlerFunc
}

var routes []route

func init() {
	routes = []route{
		{"POST", seg("/repos/{owner}/{repo}/issues/{issue_number}/comments"), wrap(handlers.CreateIssueComment)},
		{"GET", seg("/repos/{owner}/{repo}/issues/{issue_number}/comments"), wrap(handlers.ListIssueComments)},
		{"PATCH", seg("/repos/{owner}/{repo}/check-runs/{check_run_id}"), wrap(handlers.UpdateCheckRun)},
		{"POST", seg("/repos/{owner}/{repo}/check-runs"), wrap(handlers.CreateCheckRun)},
		{"GET", seg("/repos/{owner}/{repo}/pulls/{pull_number}"), wrap(handlers.GetPull)},
		{"GET", seg("/repos/{owner}/{repo}/pulls"), wrap(handlers.ListPulls)},
		{"POST", seg("/repos/{owner}/{repo}/issues"), wrap(handlers.CreateIssue)},
		{"GET", seg("/repos/{owner}/{repo}/commits/{sha}"), wrap(handlers.GetCommit)},
		{"GET", seg("/repos/{owner}/{repo}/labels"), wrap(handlers.ListLabels)},
	}
}

func seg(path string) []string {
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}

func wrap(h handlerFunc) handlerFunc { return h }

func dispatch(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return
	}

	pathSegs := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")

	for _, rt := range routes {
		if rt.method != "*" && rt.method != r.Method {
			continue
		}
		if params, ok := match(rt.segments, pathSegs); ok {
			rt.handler(w, r, params)
			return
		}
	}

	handlers.Unsupported(w, r)
}

func match(pattern, path []string) (map[string]string, bool) {
	if len(pattern) != len(path) {
		return nil, false
	}
	params := make(map[string]string)
	for i, seg := range pattern {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			params[seg[1:len(seg)-1]] = path[i]
		} else if seg != path[i] {
			return nil, false
		}
	}
	return params, true
}
