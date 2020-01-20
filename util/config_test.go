package util

import (
	"fmt"
	"github.com/zvchain/zvchain/middleware/types"
	"sync/atomic"
	"testing"
)

func tt(){
	s1 := atomic.Value{}

	var sss  *types.BlockHeader
	s1.Store(sss)
	s1 = atomic.Value{}


	ddd := s1.Load()

	if ddd.(*types.BlockHeader).Height == 2{
		fmt.Println("....")
	}
}

func TestCreatePool2(t *testing.T) {
	for i:=0 ;i < 10000000;i++{
		tt()
	}
	fmt.Printf("ok")
	//time.Sleep(1111111111111111)
}
