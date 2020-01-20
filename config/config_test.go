package config

import (
	"github.com/zvchain/zvcgo"
	"testing"
)

func TestGetNodes(t *testing.T) {
	keyBag := zvcgo.NewKeyBag()
	keyBag.ImportPrivateKeyFromString("0xc9675836d7924d6879d68f8080a6f31ab07119864e4566cdcef75bad45f6c021")
	api.SetSigner(keyBag)
	n := Node{}
	n.sk = "0xc9675836d7924d6879d68f8080a6f31ab07119864e4566cdcef75bad45f6c021"
	n.Addr = "zv33b4cfc2bcb97172dfd79339d8b67373959c27dcfdf78da56f99e627287d63e5"
	n.MinerApply(1, 1)
}
