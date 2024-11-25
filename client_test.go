package main

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type UserData struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type UsersData struct {
	UsersSlice []UserData `xml:"row"`
}

func SearchServer(w http.ResponseWriter, r *http.Request) {

	queryParams := r.URL.Query()
	limit, err := strconv.Atoi(queryParams.Get("limit"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	offset, err := strconv.Atoi(queryParams.Get("offset"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := queryParams.Get("query")
	orderField := queryParams.Get("order_field")

	orderBy, err := strconv.Atoi(queryParams.Get("order_by"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, err := os.Open("dataset.xml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	usersData := UsersData{}
	if err = xml.NewDecoder(file).Decode(&usersData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var foundUsers []User
	for _, userData := range usersData.UsersSlice {
		userName := userData.FirstName + userData.LastName
		if strings.Contains(userName, query) || strings.Contains(userData.About, query) {
			user := User{Id: userData.Id, Name: userName, Age: userData.Age, About: userData.About, Gender: userData.Gender}
			foundUsers = append(foundUsers, user)
		}
	}

	if orderBy == OrderByAsc {
		switch orderField {
		case "Name", "":
			sort.Slice(foundUsers, func(i, j int) bool {
				return foundUsers[i].Name < foundUsers[j].Name
			})

		case "Id":
			sort.Slice(foundUsers, func(i, j int) bool {
				return foundUsers[i].Id < foundUsers[j].Id
			})
		case "Age":
			sort.Slice(foundUsers, func(i, j int) bool {
				return foundUsers[i].Age < foundUsers[j].Age
			})
		default:
			http.Error(w, ErrorBadOrderField, http.StatusBadRequest)
			return
		}
	} else if orderBy == OrderByDesc {
		switch orderField {
		case "Name", "":
			sort.Slice(foundUsers, func(i, j int) bool {
				return foundUsers[i].Name > foundUsers[j].Name
			})
		case "Id":
			sort.Slice(foundUsers, func(i, j int) bool {
				return foundUsers[i].Id > foundUsers[j].Id
			})
		case "Age":
			sort.Slice(foundUsers, func(i, j int) bool {
				return foundUsers[i].Age > foundUsers[j].Age
			})
		default:
			http.Error(w, ErrorBadOrderField, http.StatusBadRequest)
			return
		}
	} else if orderBy != OrderByAsIs {
		http.Error(w, "invalid value of the OrderBy field", http.StatusBadRequest)
		return
	}

	if offset < len(foundUsers) {
		foundUsers = foundUsers[offset:]
	}

	if limit < len(foundUsers) {
		foundUsers = foundUsers[:limit]
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(foundUsers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type TestCase struct {
	Request  SearchRequest
	Response *SearchResponse
	IsError  bool
}

func TestSearchCheckout(t *testing.T) {
	cases := []TestCase{
		TestCase{
			Request: SearchRequest{
				Limit:      1,
				Offset:     0,
				OrderField: "Id",
				OrderBy:    OrderByAsc,
				Query:      "Boyd",
			},
			Response: &SearchResponse{
				Users: []User{User{
					Id:     0,
					Name:   "BoydWolf",
					Age:    22,
					About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
					Gender: "male",
				}},
				NextPage: false,
			},
			IsError: false,
		},
		TestCase{
			Request: SearchRequest{
				Limit:      -1,
				Offset:     0,
				OrderField: "Id",
				OrderBy:    OrderByAsc,
				Query:      "Boyd",
			},
			Response: nil,
			IsError:  true,
		},
		TestCase{
			Request: SearchRequest{
				Limit:      1,
				Offset:     -1,
				OrderField: "Id",
				OrderBy:    OrderByAsc,
				Query:      "Boyd",
			},
			Response: nil,
			IsError:  true,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		searchClient := SearchClient{
			URL:         testServer.URL,
			AccessToken: "",
		}
		result, err := searchClient.FindUsers(item.Request)
		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Response, result) {
			t.Errorf("[%d] wrong result\n expected %#v\n got %#v", caseNum, item.Response, result)
		}
	}
}
