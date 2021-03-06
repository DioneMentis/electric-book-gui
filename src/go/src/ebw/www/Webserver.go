package www

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/uniplaces/carbon"
	"golang.org/x/net/webdav"

	"ebw/api/jsonrpc"
	"ebw/print"
	// "ebw/util"
)

func webdavRoutes(r *mux.Router, prefix string) {
	handler := &webdav.Handler{
		Prefix:     prefix,
		FileSystem: NewOSFileSystem("test"),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			glog.Infof("%s %s : %v", r.Method, r.RequestURI, err)
		},
	}
	r.HandleFunc(prefix+"{t:.*}", func(w http.ResponseWriter, r *http.Request) {
		glog.Infof("webdav handler: %s %s", r.Method, r.RequestURI)
		handler.ServeHTTP(w, r)
	})
}

func WebError(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func RunWebServer(bind string) error {
	// Initialize the sessions manager
	if err := initSessions(); nil != err {
		return err
	}
	r := mux.NewRouter()
	r.Handle(`/`, WebHandler(repoList))
	webdavRoutes(r, `/webdav`)
	print.PrintRoutes(r)
	r.HandleFunc(`/rpc/API/json`, jsonrpc.HttpHandlerFunc)
	r.HandleFunc(`/rpc/API/json/ws`, jsonrpc.WsHandlerFunc)
	r.Handle(`/github/login`, WebHandler(githubLogin))
	r.Handle(`/github/auth`, WebHandler(githubAuth))
	r.Handle(`/github/token/{token}`, WebHandler(githubSetToken))

	r.Handle(`/github/create/fork`, WebHandler(githubCreateFork))
	r.Handle(`/github/create/new`, WebHandler(githubCreateNew))
	r.Handle(`/github/invite/{id}`, WebHandler(githubInvitationAcceptOrDecline))
	r.Handle(`/repo/{repoOwner}/{repoName}/update`, WebHandler(repoUpdate))
	r.Handle(`/repo/{repoOwner}/{repoName}/`, WebHandler(repoView))
	r.Handle(`/repo/{repoOwner}/{repoName}/commit`, WebHandler(repoCommit))
	r.Handle(`/repo/{repoOwner}/{repoName}/detail`, WebHandler(repoDetails))
	r.Handle(`/repo/{repoOwner}/{repoName}/diff`, WebHandler(repoDiff))
	r.Handle(`/repo/{repoOwner}/{repoName}/diff/{fromOID}/{toOID}/{index}`, WebHandler(repoDiffPatch))
	r.Handle(`/repo/{repoOwner}/{repoName}/diff/{fromOID}/{toOID}`, WebHandler(repoDiffFiles))
	r.Handle(`/repo/{repoOwner}/{repoName}/diff-serve/{OID}`, WebHandler(repoDiffFileServer))
	r.Handle(`/repo/{repoOwner}/{repoName}/diff-dates`, WebHandler(repoDiffDates))
	r.Handle(`/repo/{repoOwner}/{repoName}/diff-diff/{fromOID}/{toOID}`, WebHandler(repoDiffDiff))
	r.Handle(`/repo/{repoOwner}/{repoName}/files`, WebHandler(repoFileViewer))
	r.Handle(`/repo/{repoOwner}/{repoName}/merge/{remote}`, WebHandler(repoMergeRemote))
	r.Handle(`/repo/{repoOwner}/{repoName}/merge/{remote}/{branch}`, WebHandler(repoMergeRemoteBranch))
	r.Handle(`/repo/{repoOwner}/{repoName}/pull/new`, WebHandler(pullRequestCreate))
	r.Handle(`/repo/{repoOwner}/{repoName}/pull/{number}`, WebHandler(pullRequestMerge))
	r.Handle(`/repo/{repoOwner}/{repoName}/pull/{number}/close`, WebHandler(pullRequestClose))
	r.Handle(`/repo/{repoOwner}/{repoName}/push/{remote}/{branch}`, WebHandler(repoPushRemote))
	r.Handle(`/repo/{repoOwner}/{repoName}/status`, WebHandler(repoStatus))

	r.Handle(`/error/generate`, WebHandler(func(c *Context) error {
		return fmt.Errorf(`This is an error that I'm auto-generating`)
	}))
	r.Handle(`/error/report`, WebHandler(errorReporter))

	r.Handle(`/repo/{repoOwner}/{repoName}/conflict`, WebHandler(repoConflict))
	r.Handle(`/repo/{repoOwner}/{repoName}/conflict/abort`, WebHandler(repoConflictAbort))
	r.Handle(`/repo/{repoOwner}/{repoName}/conflict/resolve`, WebHandler(repoConflictResolve))

	r.Handle(`/www/{path:.*}`, WebHandler(repoFileServer))
	r.Handle(`/www-version/{version}/{repoOwner}/{repoName}/{path:.*}`,
		WebHandler(repoVersionedFileServer))
	r.Handle(`/jekyll/{repoOwner}/{repoName}/{path:.*}`, WebHandler(jekyllRepoServer))
	r.Handle(`/jekyll-restart/{repoOwner}/{repoName}/{path:.*}`, WebHandler(jekyllRepoServerRestart))

	r.Handle(`/logoff`, WebHandler(LogoffHandler))
	r.Handle(`/to-github`, WebHandler(ToGithubHandler))

	r.Handle(`/{p:.*}`, http.FileServer(http.Dir(`public`)))

	http.HandleFunc(`/`, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add(`Cache-Control`, `no-cache, no-store, must-revalidate`)
		w.Header().Add(`Pragma`, `no-cache`)
		w.Header().Add(`Expires`, `0`)
		r.ServeHTTP(w, req)
	})
	// @TODO convert to handle signals and clean shutdown
	glog.Infof("Listening on %s", bind)
	return http.ListenAndServe(bind, nil)
}

// pathRepoEdit returns the URL path to edit the given repo
func pathRepoEdit(c *Context, repoUser, repoName string) (string, error) {
	return fmt.Sprintf(`/repo/%s/%s/update`, repoUser, repoName), nil
}

func Render(w http.ResponseWriter, r *http.Request, tmpl string, data interface{}) error {
	t := template.New("").Funcs(map[string]interface{}{
		"Rand": func() string {
			return fmt.Sprintf("%d-%d", time.Now().Unix(), rand.Int())
		},
		"json": func(in interface{}) string {
			raw, err := json.Marshal(in)
			if nil != err {
				return err.Error()
			}
			return string(raw)
		},
		`JS`: func(in string) template.JS {
			return template.JS(in)
		},
		`JSStr`: func(in string) template.JSStr {
			return template.JSStr(in)
		},
		"humantime": func(in interface{}) string {
			t, ok := in.(time.Time)
			if !ok {
				return "NOT time.Time"
			}
			ct := carbon.NewCarbon(t)
			// ct = carbon.Now().SubMinutes(20)
			s, err := ct.DiffForHumans(nil, false, false, false)
			if nil != err {
				return err.Error()
			}
			return s
		},
		"raw": func(in string) template.HTML {
			return template.HTML(in)
		},
		"IsSpecialUser": func(username string) bool {
			return "craigmj" == username || "arthurattwell" == username
		},

	})
	if err := filepath.Walk("public", func(name string, info os.FileInfo, err error) error {
		// glog.Infof("walk: %s", name)
		if nil != err {
			return err
		}
		// We don't parse html in bower_components
		if strings.Contains(name, `bower_components/`) || filepath.Base(name)==`bower_components` {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(name, ".html") {
			return nil
		}
		// glog.Infof("Found template: %s", name)
		raw, err := ioutil.ReadFile(name)
		if nil != err {
			return err
		}
		if _, err := t.New(name[7:]).Parse(string(raw)); nil != err {
			glog.Errorf(`ERROR PARSING TEMPLATE %s: %s`, name, err.Error())
			return err
		}
		return nil
	}); nil != err {
		return err
	}
	if err := t.ExecuteTemplate(w, tmpl, data); nil != err {
		return err
	}
	return nil
}
