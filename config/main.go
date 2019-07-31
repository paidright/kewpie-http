package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/davidbanham/required_env"
)

var QUEUES []string
var KEWPIE_BACKEND string
var PORT int

func init() {
	required_env.Ensure(map[string]string{
		"KEWPIE_BACKEND": "",
		"PORT":           "",
	})

	for _, env := range os.Environ() {
		if strings.Contains(env, "KEWPIE_QUEUE_") {
			QUEUES = append(QUEUES, strings.Split(env, "=")[1])
		}
	}

	if len(QUEUES) == 0 {
		fmt.Println("ERROR no queues configured. Set env vars in the form KEWPIE_QUEUE_FOO=foo_bar")
	}

	fmt.Printf("INFO handling queues: %+v \n", QUEUES)

	var err error

	PORT, err = strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		fmt.Println("ERROR PORT is not a valid integer")
		panic(err)
	}

	KEWPIE_BACKEND = os.Getenv("KEWPIE_BACKEND")
}
