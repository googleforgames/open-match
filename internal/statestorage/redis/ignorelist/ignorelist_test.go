package ignorelist

import (
	"testing"

	"github.com/rafaeljusto/redigomock"
)

func TestCreate(t *testing.T) {
	pids := []string{"test1", "test2", " test3"}
	ilid := "testil"
	redisConn := redigomock.NewConn()

	redisConn.GenericCommand("ZADD").Expect("ok")
	err := Create(redisConn, ilid, pids)
	if err != nil {
		t.Fatal(err)
	}
}
