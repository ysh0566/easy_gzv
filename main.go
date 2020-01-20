package main

import (
	"easy_gzv/config"
	"easy_gzv/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/zvchain/zvchain/common"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var accounts = Accounts{
	"admin": ReadPassword(),
}

func ReadPassword() string {
	b, err := ioutil.ReadFile("password")
	if err != nil {
		return "123456"
	} else {
		return string(b)
	}
}

type Gzv struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Height  int64  `json:"height"`
	config.Node
}

func GetAllGzv(c *gin.Context) {
	gzvs := make([]Gzv, 0)
	for i := 1; i <= len(config.Nodes); i++ {
		name := fmt.Sprintf("gzv%d", i)
		g := Gzv{
			Name:    name,
			Running: util.GetProcessStatus(name),
			Height:  int64(util.GetHeight(name)),
			Node:    *config.Nodes[name],
		}
		gzvs = append(gzvs, g)
	}
	c.JSON(200, gzvs)
}

func Process(c *gin.Context) {
	name := Trim(c, "name")
	iszero := Trim(c, "iszero")
	if iszero == "0" {
		util.StopProcess(name)
	} else {
		util.StartProcess(name)
	}
	c.JSON(200, gin.H{
		"status": "commited",
	})
}

func ChangeGather(c *gin.Context) {
	name := Trim(c, "name")
	iszero := Trim(c, "iszero")
	node := config.GetNode(name)
	if node == nil {
		c.JSON(200, gin.H{
			"status": "not found",
		})
		return
	}
	if iszero == "0" {
		node.Gether = false
	} else {
		node.Gether = true
	}
	node.Save()
	c.JSON(200, gin.H{
		"status": "commited",
	})
}

func ChangeGatherAddr(c *gin.Context) {
	name := Trim(c, "name")
	addr := Trim(c, "addr")
	node := config.GetNode(name)
	if node == nil || !common.ValidateAddress(addr) {
		c.JSON(200, gin.H{
			"status": "not found",
		})
		return
	}
	node.GetherAddr = addr
	node.Save()
	c.JSON(200, gin.H{
		"status": "commited",
	})
}

func ChangeThreshold(c *gin.Context) {
	name := Trim(c, "name")
	s := Trim(c, "num")
	node := config.GetNode(name)
	if node == nil {
		c.JSON(200, gin.H{
			"status": "not found",
		})
		return
	}
	num, err := strconv.ParseFloat(s, 64)
	if num > 100000 {
		c.JSON(200, gin.H{
			"status": "error num",
		})
		return
	}
	if err != nil {
		c.JSON(200, gin.H{
			"status": err.Error(),
		})
		return
	}
	node.Threshold = uint64(num * 1000000000)
	node.Save()
	c.JSON(200, gin.H{
		"status": "commited",
	})
}

func ChangeUnfreeze(c *gin.Context) {
	name := Trim(c, "name")
	iszero := Trim(c, "iszero")
	node := config.GetNode(name)
	if node == nil {
		c.JSON(200, gin.H{
			"status": "not found",
		})
		return
	}
	if iszero == "0" {
		node.Unfreeze = false
	} else {
		node.Unfreeze = true
	}
	node.Save()
	c.JSON(200, gin.H{
		"status": "commited",
	})
}

func ChangePassword(c *gin.Context) {
	password := Trim(c, "password")
	if password == "" {
		c.JSON(200, gin.H{
			"status": "empty password",
		})
		return
	}
	accounts["admin"] = password
	_ = ioutil.WriteFile("password", []byte(password), 0666)
	c.JSON(200, gin.H{
		"status": "commited",
	})
}

func GetPrivateKey(c *gin.Context) {
	name := Trim(c, "name")
	node := config.GetNode(name)
	if node == nil {
		c.JSON(200, gin.H{
			"status": "node not found",
		})
		return
	}
	s := node.Sk()
	c.JSON(200, gin.H{
		"status": s,
	})
}

func StakeAdd(c *gin.Context) {
	name := Trim(c, "name")
	s := Trim(c, "num")
	t := Trim(c, "type")
	node := config.GetNode(name)
	if node == nil {
		c.JSON(200, gin.H{
			"status": "not found",
		})
		return
	}
	num, err := strconv.ParseFloat(s, 64)
	if err != nil {
		c.JSON(200, gin.H{
			"status": err.Error(),
		})
		return
	}
	var mType byte
	if t == "1" {
		mType = 1
	}
	node.MinerApply(uint64(num*1000000000), mType)
	c.JSON(200, gin.H{
		"status": "commited",
	})
}

func Trim(c *gin.Context, key string) string {
	v := c.Param(key)
	return strings.Trim(v, " ")
}

func main() {
	time.Sleep(time.Second * 10)
	config.Init()
	s := strings.ReplaceAll(html, "fuck", "`")
	r := gin.Default()
	r.Use(Cors())
	authorized := r.Group("", BasicAuth(accounts))
	authorized.GET("/gzvs", GetAllGzv)
	authorized.GET("/process/:name/:iszero", Process)
	authorized.GET("/gather/:name/:iszero", ChangeGather)
	authorized.GET("/gatheraddr/:name/:addr", ChangeGatherAddr)
	authorized.GET("/threshold/:name/:num", ChangeThreshold)
	authorized.GET("/unfreeze/:name/:iszero", ChangeUnfreeze)
	authorized.GET("/password/:password", ChangePassword)
	authorized.GET("/private/:name", GetPrivateKey)
	authorized.GET("/stake/:name/:type/:num", StakeAdd)
	authorized.GET("", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, s)
	})
	r.Run("0.0.0.0:9999") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

// 处理跨域请求,支持options访问
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		// 处理请求
		c.Next()
	}
}

var html = `<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="${item.name}viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <meta http-equiv="pragma" content="no-cache">
    <meta http-equiv="Cache-Control" content="no-cache, must-revalidate">
    <meta http-equiv="expires" content="0">
    <title>node</title>
    <script src="https://cdn.bootcss.com/jquery/3.4.1/jquery.min.js"></script>
</head>
<style>
    body {
        margin: 0;
        padding: 0;
    }

    * {
        box-sizing: border-box;
    }

    .node_list {
        list-style: none;
        padding: 20px;
    }

    .btn {
        margin: 20px;
    }

    .node_list li {
        display: flex;
        padding: 20px;
        width: 100%;
        margin: 10px;
        flex-direction: column;
        border-radius: 20px;
        border: 1px solid #333;

    }

    .node_list li h3 {
        font-size: 16px;
        margin: 5px;
    }

    .node_list li h3 span {
        color: #999;
        font-size: 13px;
        width: 8%;
        text-align: right;
        display: inline-block;
    }

    input {
        border: 1px solid #ddd;
        outline: none;
        padding: 4px 5px;
        line-height: 16px;
        border-radius: 2px;
    }

    input[type="text"] {
        width: 40%;
        margin-left: 10px;
    }

    button {
        background: green;
        outline: none;
        font-size: 14px;
        padding: 5px 10px;
        color: #ddd;
        border-radius: 4px;
    }
</style>

<body>
<div class="content">
    <button class="btn" onclick="funPassword()">修改密码</button>
    <ul class="node_list">
    </ul>

</div>
</body>
<script>

    updata()
    function funcStake(name) {
        let stakeValue = $(fuck#${name}addValuefuck).val();
        var radios = document.getElementsByName("${item.name}add");
        let type = $(fuckinput[name='${name}add']:checkedfuck).val();
        $.ajax({
            type: "get",
            // cache: false,
            url: fuck/stake/${name}/${type}/${stakeValue}fuck,
            dataType: "JSON",
            success: (response) => {
                window.location.reload()
            }
        });
    }
    function funExport(name) {
        $.ajax({
            type: "get",
            // cache: false,
            url: fuck/private/${name}fuck,
            dataType: "JSON",
            success: (response) => {
                alert(response.status)
            }
        });
    }
    function funPassword() {
        var passwordValue = window.prompt("修改密码")
        if (!passwordValue) { return }
        $.ajax({
            type: "get",
            cache: false,
            url: fuck/password/${passwordValue}fuck,
            dataType: "JSON",
            success: (response) => {
                window.location.reload()
            }
        });
    }
    function funcRunning(name, iszero) {
        $.ajax({
            type: "get",
            cache: false,
            url: fuck/process/${name}/${iszero}fuck,
            dataType: "JSON",
            success: (response) => {
                updata()
            }
        });
    }
    function funRadioGether(name, iszero) {
        $.ajax({
            type: "get",
            cache: false,
            url: fuck/gather/${name}/${iszero}fuck,
            dataType: "JSON",
            success: (response) => {
                updata()
            }
        });
    }
    function funGatheraddr(name) {
        let addr = $(fuck#${name}gatheraddrfuck).val();
        $.ajax({
            type: "get",
            cache: false,
            url: fuck/gatheraddr/${name}/${addr}fuck,
            dataType: "JSON",
            success: (response) => {
                updata()
            }
        });
    }
    function funThreshold(name) {
        let num = $(fuck#${name}hresholdfuck).val();
        $.ajax({
            type: "get",
            cache: false,
            url: fuck/threshold/${name}/${num}fuck,
            dataType: "JSON",
            success: (response) => {
                updata()
            }
        });
    }
    function funcUnfreezeA(name, iszero) {

        $.ajax({
            type: "get",
            cache: false,
            url: fuck/unfreeze/${name}/${iszero}fuck,
            dataType: "JSON",
            success: (response) => {
                updata()
            }
        });
    }

    function updata() {
        $.ajax({
            type: "get",
            cache: false,
            url: "/gzvs",
            dataType: "JSON",
            success: (response) => {
                console.log(response)
                let html = ''
                response.forEach(item => {
                    let node_radio = ''
                    let gether = '' //归结
                    let unfreeze = ''
                    if (item.running) { // 节点开关
                        node_radio = fuck
                <h3>
                <span>启动节点：</span>

                    <label for="ee">
                      <input type="radio" name="${item.name}radioName" checked="checked"  onclick="funcRunning('${item.name}','1')"> 开
                        <input type="radio" name="${item.name}radioName"   onclick="funcRunning('${item.name}','0')"> 关
                    </label>

                  </h3>fuck
                    } else {
                        node_radio = fuck
                <h3>
                    <span>启动节点：</span>
                    <input type="radio" name="${item.name}radioName" checked="checked"  onclick="funcRunning('${item.name}','1')"> 开
                    <input type="radio" name="${item.name}radioName"  checked="checked"   onclick="funcRunning('${item.name}','0')"> 关
                  </h3>fuck
                    }
                    if (item.gether) {
                        gether = fuck
                <input type="radio" name="${item.name}gether" checked="checked"  onclick="funRadioGether('${item.name}','1')"> 开
                    <input type="radio" name="${item.name}gether"   onclick="funRadioGether('${item.name}','0')"> 关fuck
                    } else {
                        gether = fuck
              <input type="radio" name="${item.name}gether"   onclick="funRadioGether('${item.name}','1')"> 开
                  <input type="radio" name="${item.name}gether" checked="checked"   onclick="funRadioGether('${item.name}','0')"> 关fuck
                    }

                    if (item.unfreeze) {
                        unfreeze = fuck

                <input type="radio" name="${item.name}unfreeze" checked="checked"  onclick="funcUnfreezeA('${item.name}','1')" > 开
                <input type="radio" name="${item.name}unfreeze"  onclick="funcUnfreezeA('${item.name}','0')" > 关
           fuck
                    } else {
                        unfreeze = fuck
                <input type="radio" name="${item.name}unfreeze"  onclick="funcUnfreezeA('${item.name}','1')"> 开
                <input type="radio" name="${item.name}unfreeze"  checked="checked"  onclick="funcUnfreezeA('${item.name}','0')"> 关
            fuck
                    }


                    html += fuck
            <li>
              <h3>
                <span> 节点名称：</span> ${item.name}
              </h3>
              <h3>
                <span> 节点块高：</span> ${item.height}
              </h3>
              <h3>
                <span> 节点地址：</span> ${item.addr}
              </h3>
                ${node_radio}
              <h3>

              <span>  归结门限：</span>
                <input type="text" name="${item.name}" value="${item.threshold / 1e9}" id="${item.name}hreshold"> <button onclick="funThreshold('${item.name}')">确认修改</button>
              </h3>
              <h3>

                <span>归结地址: </span>
                ${gether}
                <input type="text" name="${item.name}" value='${item.gether_addr}' id="${item.name}gatheraddr"> <button onclick="funGatheraddr('${item.name}')">确认</button>
              </h3>
              <h3>
                <span> 自动解冻开关：</span>
              ${unfreeze}
              </h3>
              <h3>
              <span>增加质押：</span>
              <input type="radio" name="${item.name}add" checked="checked" value='0'  id="">
                验证
              <input type="radio" name="${item.name}add" value='1'  id="">
              提案
              <input type='text' name="" id="${item.name}addValue">
              <button onclick='funcStake("${item.name}")'>确认</button>
            </h3>


              <button class="btn" onclick="funExport('${item.name}')">导出私钥</button>
            </li>
          fuck
                })
                $('.node_list').html(html)
            }
        });
    }


</script>

</html>
`
