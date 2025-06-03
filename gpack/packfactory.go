package gpack

import (
	"sync"

	"github.com/liyee/gray/giface"
)

var pack_once sync.Once

type pack_factory struct{}

var factoryInstance *pack_factory

func Factory() *pack_factory {
	pack_once.Do(func() {
		factoryInstance = new(pack_factory)
	})

	return factoryInstance
}

func (f *pack_factory) NewPack(kind string) giface.IDataPack {
	var dataPack giface.IDataPack

	switch kind {
	// Zinx standard default packaging and unpackaging method
	// (Zinx 标准默认封包拆包方式)
	case giface.GrayDataPack:
		dataPack = NewDataPack()
	case giface.GrayDataPackOld:
		dataPack = NewDataPackLtv()
		// case for custom packaging and unpackaging methods
		// (case 自定义封包拆包方式case)
	default:
		dataPack = NewDataPack()
	}

	return dataPack
}
