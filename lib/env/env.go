package env

import (
	"encoding/json"
	"os/exec"
	"sync"
)

var (
	lock  sync.Mutex
	goEnv map[string]string
)

func GetEnv() (map[string]string, error) {
	lock.Lock()
	defer lock.Unlock()

	if goEnv != nil {
		return goEnv, nil
	}

	cmd := exec.Command("go", "env", "-json")
	data, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &goEnv); err != nil {
		return nil, err
	}

	return goEnv, nil
}
