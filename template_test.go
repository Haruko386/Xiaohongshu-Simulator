package main

import (
	"Xiaohongshu_Simulator/models"
	"html/template"
	"io"
	"testing"
)

func TestTemplatesParse(t *testing.T) {
	templates, err := template.ParseGlob("templates/*.tmpl")
	if err != nil {
		t.Fatalf("templates should parse: %v", err)
	}

	renderCases := []struct {
		name string
		data any
	}{
		{name: "index.tmpl", data: map[string]any{"Posts": []models.Post{}, "NeedsLogin": true, "LoggedInUserID": ""}},
		{name: "profile.tmpl", data: map[string]any{"User": models.User{}, "Posts": []models.Post{}, "IsOwner": false, "LoggedInUserID": "", "FollowerCount": 0, "FollowingCount": 0, "TotalFavorited": 0, "IsFollowing": false}},
		{name: "publish.tmpl", data: map[string]any{"LoggedInUserID": "1"}},
		{name: "login.tmpl", data: nil},
	}
	for _, testCase := range renderCases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := templates.ExecuteTemplate(io.Discard, testCase.name, testCase.data); err != nil {
				t.Fatalf("template should render: %v", err)
			}
		})
	}
}
