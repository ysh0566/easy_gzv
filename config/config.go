package config

import (
	"easy_gzv/util"
	"encoding/json"
	"fmt"
	ini "github.com/glacjay/goini"
	"github.com/tidwall/gjson"
	"github.com/zvchain/zvcgo"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/consensus/model"
	"github.com/zvchain/zvchain/middleware/types"
	"io/ioutil"
	"net/http"
	"os/user"
	"strings"
	"sync"
	"time"
)

var api = zvcgo.NewApi("https://api.firepool.pro:8101")
var Nodes map[string]*Node
var lock sync.Mutex

func Init() {
	Nodes = getNodes()
}

type Node struct {
	WorkDir    string       `json:"work_dir"`
	Addr       string       `json:"addr"`
	Gether     bool         `json:"gether"`
	GetherAddr string       `json:"gether_addr"`
	Threshold  uint64       `json:"threshold"`
	Unfreeze   bool         `json:"unfreeze"`
	ticker     *time.Ticker `json:"-"`
	quit       chan bool    `json:"-"`
	isRunning  bool         `json:"-"`
	sk         string
}

func GetNode(name string) *Node {
	lock.Lock()
	defer lock.Unlock()
	node, ok := Nodes[name]
	if !ok {
		return nil
	}
	return node
}

func (n *Node) Save() {
	bs, err := json.Marshal(Nodes)
	err = ioutil.WriteFile("config.json", bs, 0666)
	if err != nil {
		panic(err)
	}
}

func (n *Node) Run() {
	if n.isRunning {
		return
	}
	n.ticker = time.NewTicker(time.Minute)
	n.isRunning = true
	go func() {
		defer func() {
			n.ticker.Stop()
			n.isRunning = false
		}()
		for true {
			select {
			case <-n.ticker.C:
				go func() {
					defer func() {
						err := recover()
						if err != nil {
							fmt.Printf("execute task panic: %v", err)
						}
					}()
					err := n.ExecuteGather()
					if err != nil {
						fmt.Printf("execute task return error: %v", err)
					}
					n.ExecuteUnfreeze()
				}()
			case <-n.quit:
				return
			}
		}
	}()
}

func (n *Node) Stop() {
	if !n.isRunning {
		return
	}
	n.quit <- true
}

func (n *Node) Sk() string {
	return n.sk
}

func (n *Node) ExecuteGather() error {
	if !n.Gether || n.GetherAddr == "" {
		return nil
	}
	address, err := zvcgo.NewAddressFromString(n.Addr)
	if err != nil {
		return err
	}
	gaddr, err := zvcgo.NewAddressFromString(n.GetherAddr)
	if err != nil {
		return err
	}
	balanceF, err := api.Balance(address)
	if err != nil {
		return nil
	}
	balance := uint64(balanceF * 1000000000)
	asset, _ := zvcgo.NewAssetFromString(fmt.Sprintf("%d Ra", balance-1000000000))
	if balance >= n.Threshold+1000000000 {
		tx := zvcgo.NewTransferTransaction(address, gaddr, asset, nil)
		nonce, _ := api.GetNonce(address)
		tx.SetNonce(nonce)
		_, _ = api.SignAndSendTransaction(tx)
	}
	return nil
}

func (n *Node) ExecuteUnfreeze() {
	if n.Unfreeze {
		b, height := CheckFreeze(n.Addr)
		if b {
			currenHeight, _ := api.BlockHeight()
			address, err := zvcgo.NewAddressFromString(n.Addr)
			if err != nil {
				return
			}
			if currenHeight-1300 > uint64(height) {
				nonce, _ := api.GetNonce(address)
				abortTx := zvcgo.NewTransferTransaction(address, address, zvcgo.Asset{}, []byte{0}).RawTransaction
				abortTx.Type = 4
				abortTx.SetNonce(nonce)
				_, _ = api.SignAndSendTransaction(abortTx)

				addTx := zvcgo.NewTransferTransaction(address, address, zvcgo.Asset{}, []byte{1, 0}).RawTransaction
				addTx.Type = 3
				addTx.SetNonce(nonce + 1)
				_, _ = api.SignAndSendTransaction(addTx)
			}
		}
	}
}

func (n *Node) MinerApply(amount uint64, minerType byte) {
	if n.sk == "" {
		return
	}
	payload := MinerApplyPayload(n.sk, minerType)
	address, err := zvcgo.NewAddressFromString(n.Addr)
	if err != nil {
		return
	}
	nonce, _ := api.GetNonce(address)
	asset, _ := zvcgo.NewAssetFromString(fmt.Sprintf("%d Ra", amount))
	abortTx := zvcgo.NewTransferTransaction(address, address, asset, payload).RawTransaction
	abortTx.Type = 3
	abortTx.SetNonce(nonce)
	_, _ = api.SignAndSendTransaction(abortTx)
}

func getNodes() map[string]*Node {
	oldNodes := make(map[string]*Node)
	newNodes := make(map[string]*Node)
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadFile("config.json") // just pass the file name
	if err == nil {
		err := json.Unmarshal(b, &oldNodes)
		if err != nil {
			panic(err)
		}
	}
	files, err := ioutil.ReadDir(supervisorPath)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if strings.Contains(file.Name(), "gzv") {
			key := strings.ReplaceAll(file.Name(), ".ini", "")
			index := strings.ReplaceAll(key, "gzv", "")
			workDir := fmt.Sprintf("%s/easy_gzv/gzv_run%s", u.HomeDir, index)
			newNodes[key] = &Node{
				WorkDir: workDir,
				Addr:    GetAddr(workDir + "/zv.ini"),
				quit:    make(chan bool),
			}
			if _, ok := oldNodes[key]; ok {
				newNodes[key].Unfreeze = oldNodes[key].Unfreeze
				newNodes[key].Gether = oldNodes[key].Gether
				newNodes[key].GetherAddr = oldNodes[key].GetherAddr
				newNodes[key].Threshold = oldNodes[key].Threshold
			}
			newNodes[key].Run()
		}
	}
	Import(newNodes)
	bs, err := json.Marshal(newNodes)
	err = ioutil.WriteFile("config.json", bs, 0666)
	if err != nil {
		panic(err)
	}
	return newNodes
}

func Import(ns map[string]*Node) {
	keyBag := zvcgo.NewKeyBag()
	for _, node := range ns {
		privateKey, err := util.GetPrivateKey(fmt.Sprintf("%s/keystore", node.WorkDir),
			node.Addr, GetPassword(fmt.Sprintf("%s/miner.sh", node.WorkDir)))
		if err != nil {
			panic(err)
		}
		err = keyBag.ImportPrivateKeyFromString(privateKey)
		if err != nil {
			panic(err)
		}
		node.sk = privateKey
	}
	api.SetSigner(keyBag)
}

func GetAddr(file string) string {
	d, err := ini.Load(file)
	if err != nil {
		return ""
	}
	s, _ := d.GetString("gzv", "miner")
	return s
}

func GetPassword(file string) string {
	bs, _ := ioutil.ReadFile(file)
	ss := strings.Split(string(bs), "--password ")
	if len(ss) < 2 {
		return ""
	}
	res := strings.Trim(ss[1], " ")
	res = strings.Trim(ss[1], "\n")
	return res
}

func CheckFreeze(addr string) (bool, int64) {
	data := fmt.Sprintf(`{
    "method": "Gzv_minerInfo",
    "params": [
         "%s", ""
    ],
    "jsonrpc": "2.0",
    "id": 1
}`, addr)
	reader := strings.NewReader(data)
	resp, err := http.Post("https://api.firepool.pro:8101", "application/json", reader)
	if err != nil {
		return false, 0
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	ress := gjson.GetBytes(body, "result.overview").Array()
	for _, res := range ress {
		if res.Get("type").String() == "verify node" {
			if res.Get("miner_status").String() == "frozen" {
				update := res.Get("status_update_height").Int()
				return true, update
			}
		}
	}
	return false, 0
}

func MinerApplyPayload(skString string, minerType byte) []byte {
	sk := &common.PrivateKey{}
	sk.ImportKey(common.FromHex(skString))
	minerDO, err := model.NewSelfMinerDO(sk)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	var bpk groupsig.Pubkey
	bpk.SetHexString(minerDO.PK.GetHexString())
	pks := &types.MinerPks{
		MType: types.MinerType(minerType),
	}
	pks.Pk = bpk.Serialize()
	pks.VrfPk = base.Hex2VRFPublicKey(minerDO.VrfPK.GetHexString())
	data, err := types.EncodePayload(pks)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return data
}
