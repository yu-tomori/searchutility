package mediainfo

import (
	"encoding/json"
	"fmt"
	"os"
)

type MediaInfo struct {
	Name       string  `json:"name"`
	DomainRank float64 `json:"domain_rank"`
	UniqueUser float64 `json:"unique_user"`
}

func (mi MediaInfo) String() string {
	return fmt.Sprintf("DR:%.1f, UU:%.1f\n%s",
		mi.DomainRank, mi.UniqueUser, mi.Name)
}

var MediaMap map[string]MediaInfo

func init() {
	f := "./mediainfo.json"
	if _, err := os.Stat(f); err != nil {
		e := fmt.Errorf("%v: %s is not existent.", err, f)
		panic(e)
		os.Exit(1)
	}

	file, err := os.Open(f)
	if err != nil {
		e := fmt.Errorf("%v: cannot open %s", err, f)
		panic(e)
		os.Exit(1)
	}
	defer file.Close()

	// load from json to b
	fileInfo, err := file.Stat()
	if err != nil {
		panic(err)
	}
	b := make([]byte, fileInfo.Size())
	if _, err := file.Read(b); err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &MediaMap)
	if err != nil {
		panic(err)
	}
}
