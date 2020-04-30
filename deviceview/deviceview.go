package deviceview

import (
	"io"
	"lwmq/dispatcher"
	"net/http"
	"sort"
	"strconv"
	"text/template"

	"github.com/labstack/echo"
)

type deviceInfo struct {
	Odd      int
	ClientID string
	Sublist  string
	CreateT  string
	Online   int
}

// Template template
type Template struct {
	templates *template.Template
}

// Render render
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

var templates *Template

func devicelist(c echo.Context) error {

	var sceneList []string
	var deviceList []*deviceInfo

	for k := range dispatcher.Mserver.Mclients {
		sceneList = append(sceneList, k)
	}
	sort.Strings(sceneList)

	for idx, k := range sceneList {

		v := dispatcher.Mserver.Mclients[k]
		devinfo := &deviceInfo{
			ClientID: k,
			CreateT:  v.CreateTime,
		}

		showData := ""
		if v.SubList.Len() > 0 {
			for j := v.SubList.Front(); j != nil; j = j.Next() {
				subscribe := j.Value.(*dispatcher.SubTopic)
				showData += "Topic:" + subscribe.Topic + "   Qos:" + strconv.Itoa(int(subscribe.Qos)) + "<br/>"
			}
		}
		devinfo.Sublist = showData

		if (idx % 2) == 0 {
			devinfo.Odd = 1
		} else {
			devinfo.Odd = 0
		}

		if v.Status == dispatcher.Connected {
			devinfo.Online = 1
		} else {
			devinfo.Online = 0
		}

		deviceList = append(deviceList, devinfo)
	}

	return c.Render(http.StatusOK, "devices", deviceList)
}

func getdevices() {
	e := echo.New()
	e.Static("/", "html")

	e.Renderer = templates

	e.GET("/", devicelist)

	e.Start(":1888")
}

// Startservice start http service
func Startservice() {
	go getdevices()
}

func init() {
	templates = &Template{
		templates: template.Must(template.ParseGlob("html/*.html")),
	}
}
