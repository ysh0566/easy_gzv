package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kolo/xmlrpc"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/zvchain/zvcgo"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/storage/tasdb"
	"golang.org/x/crypto/scrypt"
	"strconv"
	"strings"
)

type KeyStoreRaw struct {
	Key     []byte
	IsMiner bool
}

func GetPrivateKey(ks string, addr string, password string) (string, error) {
	options := &opt.Options{
		OpenFilesCacheCapacity:        10,
		WriteBuffer:                   8 * opt.MiB, // Two of these are used internally
		Filter:                        filter.NewBloomFilter(10),
		CompactionTableSize:           2 * opt.MiB,
		CompactionTableSizeMultiplier: 2,
	}
	db, err := tasdb.NewLDBDatabase(ks, options)
	if err != nil {
		return "", err
	}
	defer db.Close()
	v, err := db.Get([]byte(addr))
	if err != nil {
		return "", fmt.Errorf("your address %s not found in your keystore directory", addr)
	}

	salt := common.Sha256([]byte(password))
	scryptPwd, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
	if err != nil {
		return "", err
	}

	bs, err := common.DecryptWithKey(scryptPwd, v)
	if err != nil {
		return "", err
	}

	var ksr = new(KeyStoreRaw)
	if err = json.Unmarshal(bs, ksr); err != nil {
		return "", err
	}

	secKey := new(common.PrivateKey)
	if !secKey.ImportKey(ksr.Key) {
		return "", errors.New("invalid data")
	}
	return common.ToHex(secKey.ExportKey()), nil
}

func GetProcessStatus(name string) bool {
	//STOPPED
	client, _ := xmlrpc.NewClient("http://127.0.0.1:9001/RPC2", nil)
	result := struct {
		Status string `xmlrpc:"statename"`
	}{}
	client.Call("supervisor.getProcessInfo", name, &result)
	if result.Status == "RUNNING" {
		return true
	} else {
		return false
	}
}

func StartProcess(name string) {
	client, _ := xmlrpc.NewClient("http://127.0.0.1:9001/RPC2", nil)
	result := struct {
	}{}
	client.Call("supervisor.startProcess", []interface{}{name, true}, &result)
}

func StopProcess(name string) {
	client, _ := xmlrpc.NewClient("http://127.0.0.1:9001/RPC2", nil)
	result := struct {
	}{}
	client.Call("supervisor.stopProcess", name, &result)
}

func GetHeight(name string) uint64 {
	numS := strings.ReplaceAll(name, "gzv", "")
	n, err := strconv.Atoi(numS)
	if err != nil {
		return 0
	}
	height, _ := zvcgo.NewApi(fmt.Sprintf("http://127.0.0.1:810%d", n)).BlockHeight()
	return height
}
