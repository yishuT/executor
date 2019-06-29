package callback

import (
	"time"
)

type Callbacker struct {
}

func NewCallbacker() *Callbacker {
	return &Callbacker{}
}

func (cb *Callbacker) Defer(cbf func(), delay time.Duration) {

}
