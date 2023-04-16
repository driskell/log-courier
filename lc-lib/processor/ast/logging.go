package ast

import "gopkg.in/op/go-logging.v1"

var log *logging.Logger

func init() {
	log = logging.MustGetLogger("processor")
}
