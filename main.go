package main

import (
	go_publish_typecho "go_publish_typecho/go-publish-typecho"
)

func main() {

	go_publish_typecho.Setup("https://your_typecho_website","login_name","login_password")
	p := go_publish_typecho.PostChapterBody{
		Title:        "这里是标题",
		Text:         "内容内容内容",
		CategoryIds:  []int{9,11,10},
		Field:        []go_publish_typecho.PostChapterAdditionalField{
			{
				Name:  "第一个类型",
				Type:  "type1",
				Value: "value1",
			},
			{
				Name:  "第二个类型",
				Type:  "type2",
				Value: "value3",
			},
			{
				Name:  "第三个类型",
				Type:  "type3",
				Value: "value3",
			},
		},
		Cid:          "",
		Markdown:     true,
		Date:         "",
		Tags:         "标签1,标签2",
		Visibility:   "1",
		Password:     "",
		AllowComment: true,
		AllowPing:    true,
		AllowFeed:    true,
		Trackback:    "",
		Do:           "publish",
		Timezone:     28800,
	}

	//publish your chapter
	go_publish_typecho.ExcuteAction(go_publish_typecho.Action_PostChapter, p)

	//keep programs stay running
	i := make(chan int)
	<- i
}
