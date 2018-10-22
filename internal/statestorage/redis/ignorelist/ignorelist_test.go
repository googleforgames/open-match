package ignorelist

import (
	"testing"

	"github.com/rafaeljusto/redigomock"
)

func TestCreate(*testing.T) {
	pids := "test1 test2 test3"
	ilid := "testil"
	redisConn := redigomock.NewConn()
	err := Create(redisConn, ilid, pids)
	if err != nil {
		panic(err.Error())
	}
}
