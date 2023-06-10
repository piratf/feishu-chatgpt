package handlers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pavel-one/EdgeGPT-Go"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type MessageActionUseEdgeGPT struct { /*消息，走 bing 接口*/
}

var edgeGPT *EdgeGPT.Storage

func InitEdgeGPTServer() {
	edgeGPT = EdgeGPT.NewStorage()
}

func asyncResponse(ctx context.Context, key string, ask string, msgId *string) {
	log.Printf("async response, key: %s, req: %s\n", key, ask)
	gpt, err := edgeGPT.GetOrSet(key)
	if err != nil {
		log.Println("get gpt failed, err: ", err)
		replyMsg(ctx, fmt.Sprintf(
			"🤖️：创建会话失败了，请稍后再试～\n错误信息: %v", err), msgId)
	}
	// 异步请求，丢到容器里，然后立即返回
	mw, err := gpt.AskSync("balanced", ask)
	if err != nil {
		log.Println("ask sync failed, err: ", err)
		replyMsg(ctx, fmt.Sprintf(
			"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), msgId)
	}
	ans := mw.Answer.GetAnswer()
	log.Println("ans: ", ans)

	err = replyMsg(ctx, ans, msgId)
	if err != nil {
		replyMsg(ctx, fmt.Sprintf(
			"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), msgId)
	}
}

func HandleEdgeGPT(c *gin.Context) {
	ask := c.PostForm("ask")
	msgId := c.PostForm("msg-id")
	key := c.PostForm("key")
	log.Printf("get edge gpt, ask: %s, msg_id: %s, key: %s", ask, msgId, key)
	asyncResponse(c, key, ask, &msgId)
}

func callEdgeGPT(a *ActionInfo) {
	// 转发请求到另一个服务，服务收到请求后转发给 bing
	requestURL := viper.GetString("EDGE_GPT_URL")
	values := make(url.Values)
	values.Set("ask", a.info.qParsed)
	values.Set("msg-id", *a.info.msgId)
	values.Set("key", *a.info.sessionId)
	res, err := http.DefaultClient.PostForm(requestURL, values)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("client: got response!\n")
	fmt.Printf("client: status code: %d\n", res.StatusCode)

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("client: response body: %s\n", resBody)
}

func (*MessageActionUseEdgeGPT) Execute(a *ActionInfo) bool {
	log.Println("call edge gpt without wait for response")
	go callEdgeGPT(a)
	// wait to make sure request has been sent
	time.Sleep(time.Second * 2)
	return true
}
