package e2e

import (
	"encoding/json"
	"io/ioutil"
)

type Stack struct {
	Members []*Member `json:"members,omitempty"`
}

type Member struct {
	ExposedFireflyPort int `json:"exposedFireflyPort,omitempty"`
}

func GetMemberPort(filename string, n int) (int, error) {
	jsonBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	var stack Stack
	err = json.Unmarshal(jsonBytes, &stack)
	if err != nil {
		return 0, err
	}

	return stack.Members[n].ExposedFireflyPort, nil
}
