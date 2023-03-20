package go_publish_typecho

import (
	"fmt"
	"github.com/gocolly/colly"
	"net/url"
	"reflect"
	"strings"
	"time"
)

type Action int
const (
	Action_Unknown	Action = iota
	Action_Login
	Action_PostChapter
)

type PostParamsRawData interface {
	ToRawString() 	string
}

type PostChapterBody struct {
	Title 			string							`json:"title"`
	Text 			string							`json:"text"`
	CategoryIds		[]int							`json:"category[]"`
	Field 			[]PostChapterAdditionalField	`json:"field"`
	Cid				string							`json:"cid"`
	Markdown 		bool							`json:"markdown"`
	Date			string							`json:"date"`
	Tags			string							`json:"tags"`
	Visibility		string							`json:"visibility"`
	Password		string							`json:"password"`
	AllowComment	bool							`json:"allowComment"`
	AllowPing		bool							`json:"allowPing"`
	AllowFeed		bool							`json:"allowFeed"`
	Trackback		string							`json:"trackback"`
	Do				string							`json:"do"`
	Timezone		int								`json:"timezone"`
}

type PostChapterAdditionalField struct {
	Name 	string		`json:"fieldNames[]"`
	Type	string		`json:"fieldTypes[]"`
	Value	string		`json:"fieldValues[]"`
}

func (p PostChapterBody)ToRawString() string  {
	res := ""
	v := reflect.ValueOf(p)
	for i := 0; i < v.NumField(); i++ {
		mapKey := getJsonStructTagForKey(&p, v.Type().Field(i).Name)

		if len(mapKey) > 0 {
			res += getStringFromInterface(v.Field(i).Interface(), mapKey)
		}
		if i < v.NumField() - 1 {
			res += "&"
		}
	}
	return res
}

func getStringFromInterface(v interface{}, mapKey string) string {
	var res string
	switch x := v.(type) {
	case int:
		res = fmt.Sprintf("%s=%d",mapKey,x)
	case bool:
		if x {
			res = fmt.Sprintf("%s=%d",mapKey,1)
		}else {
			res = fmt.Sprintf("%s=%d",mapKey,0)
		}
	case string:
		res = fmt.Sprintf("%s=%s",mapKey,x)
	case []int:
		for k, i := range x {
			res += fmt.Sprintf("%s=%d",mapKey ,i)
			if k < len(x) - 1 {
				res += "&"
			}
		}
	case []PostChapterAdditionalField:
		for k, i := range x {
			postV := reflect.ValueOf(i)
			for j := 0; j < postV.NumField(); j++ {
				mapKey := getJsonStructTagForKey(&i, postV.Type().Field(j).Name)
				res += getStringFromInterface(postV.Field(j).Interface(), mapKey)
				if j < postV.NumField() - 1 {
					res += "&"
				}
			}
			if k < len(x) - 1 {
				res += "&"
			}
		}
	default:
		res = fmt.Sprintf("%s=%s",mapKey,x)
	}
	return res
}

func getJsonStructTagForKey(body interface{}, key string) string {
	field, ok := reflect.TypeOf(body).Elem().FieldByName(key)
	if !ok {
		fmt.Println("Field not found")
		return ""
	}
	return field.Tag.Get("json")
}

type CrabParams struct {
	pageUrlPath 	string
	querySelector	string
	actionName		string
	reqHeader		map[string]string
	//if set, will ignore custom post params
	actionParams 	map[string]string
}

var crabDomain *url.URL

var c *colly.Collector

var typechoBasicCrabParams *map[Action]CrabParams

var typechoPostParams = &map[Action][]PostParamsRawData {
	Action_PostChapter: []PostParamsRawData{},
}

var countingForUrlVisited = map[string]*int{}
var maxRetryTimes = 2
var firstLoginTime time.Time

func Setup(hostUrl string, loginName string, loginPwd string)  {
	if len(loginName) <= 0 || len(loginPwd) <= 0 {
		panic("Login Username/Password is Empty.")
	}

	u, err := url.Parse(hostUrl)
	if err != nil {
		panic(err)
	}
	crabDomain = u
	setupCrabBasicParams()

	loginParams := (*typechoBasicCrabParams)[Action_Login]
	loginParams.actionParams["name"] = loginName
	loginParams.actionParams["password"] = loginPwd
	loginParams.actionParams["referer"] = ""

	setupCrab()
}

func setupCrabBasicParams()  {
	typechoBasicCrabParams = &map[Action]CrabParams{
		Action_Login: CrabParams{
			pageUrlPath: "/admin/login.php",
			querySelector:   "form[name='login']",
			actionName:    "action/login",
			reqHeader: map[string]string{
				"Referer" : crabDomain.String() + "/admin/login.php",
				"Content-Type" : "application/x-www-form-urlencoded",
			},
			actionParams: map[string]string{},
		},
		Action_PostChapter: CrabParams{
			pageUrlPath: "/admin/write-post.php",
			querySelector:   "form[name='write_post']",
			actionName:    "action/contents-post-edit",
			reqHeader: map[string]string{
				"Referer" : crabDomain.String() + "/admin/write-post.php",
				"Content-Type" : "application/x-www-form-urlencoded",
			},
		},
	}
}

func setupCrab()  {
	if c != nil {
		fmt.Println("Crab Already exsited.")
		return
	}
	// Instantiate default collector
	c = colly.NewCollector(
		colly.AllowedDomains(crabDomain.Host),
		colly.AllowURLRevisit(),
	)

	for k, v := range *typechoBasicCrabParams {
		fmt.Println("Build HTML Handler ",v)
		crabParams := v
		c.OnHTML(crabParams.querySelector, func(e *colly.HTMLElement) {
			nextAction := e.Attr("action")
			fmt.Println("Get Action :", nextAction)

			fmt.Println("action ",crabParams)
			if len(crabParams.actionParams) > 0 {
				postParams := crabParams.actionParams
				fmt.Println("Setup Installed Post Params :",postParams)

				fmt.Println("Send Data : ", postParams)
				e.Request.Post(nextAction, postParams)
			}else {
				var postParams PostParamsRawData
				postParamsList := (*typechoPostParams)[k]
				if len(postParamsList) > 0 {
					postParams = postParamsList[0]
					(*typechoPostParams)[k] = postParamsList[1:]
				}
				fmt.Println("Setup Custom Post Params :",postParams)
				fmt.Println("Send Data : ", postParams)
				e.Request.PostRaw(nextAction, []byte(postParams.ToRawString()))
			}

		})
	}

	c.OnRequest(func(request *colly.Request) {
		requestUrl := request.URL.String()
		fmt.Println("Request ",  requestUrl)
		count := countingForUrlVisited[requestUrl];
		if count == nil {
			count = new(int)
			*count = 0
			countingForUrlVisited[requestUrl] = count
			fmt.Println("Count Not init.")
		}
		*count++
		fmt.Println(requestUrl, " Counting : ",*count)
		if *count >= maxRetryTimes  {
			fmt.Println(requestUrl, " visit too much times. Please check logic for your procedure.")
			request.Abort()
			return
		}

		//TODO:can make faster without for-range
		for _, v := range *typechoBasicCrabParams {
			if strings.Contains(requestUrl ,v.actionName) {
				for i, i2 := range v.reqHeader {
					request.Headers.Set(i,i2)
				}
				fmt.Println("Set Header : ", v.reqHeader)
			}
		}
		fmt.Println("Body : ", request.Body)
	})

	c.OnResponse(func(response *colly.Response) {
		cookies := c.Cookies(response.Request.URL.String())
		fmt.Println("Cookies:", cookies)

		fmt.Println("Get Response ", response.Request.URL," Headers : ", response.Headers)

	})
	c.OnError(func(response *colly.Response, err error) {
		fmt.Println("error Resp : ", response)
		fmt.Println("error : ",err)
	})
}

func ExcuteAction(action Action, params PostParamsRawData)  {
	//check if login
	if len(c.Cookies(crabDomain.String())) <= 1 || needRelogin() {
		fmt.Println("Not Login yet, Try Login.")
		excuteAction(Action_Login)
		c.Wait()
		firstLoginTime = time.Now()
	}
	actionParamsList := (*typechoPostParams)[action]
	actionParamsList = append(actionParamsList, params)
	(*typechoPostParams)[action] = actionParamsList
	fmt.Println("Excute Action")
	excuteAction(action)
}

func excuteAction(action Action)  {
	actionParams := (*typechoBasicCrabParams)[action]
	if len(actionParams.pageUrlPath) <= 0 {
		return
	}

	c.Visit(crabDomain.String() + actionParams.pageUrlPath)
	c.Wait()
	clearTmpData()
}

func needRelogin() bool {
	//i guess cookies will be expired after 1 day
	if time.Now().Sub(firstLoginTime) <= time.Hour * 24 {
		return true
	}else {
		return false
	}
}

func clearTmpData()  {
	countingForUrlVisited = map[string]*int{}
}
