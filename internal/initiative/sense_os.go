package initiative

import (
	"os"
	"time"
)

func osReadDir(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

type fileInfo interface{ IsDir() bool }

func osStat(path string) (fileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return info, nil
}

// Machine 是这一轮从操作系统读到的状态。名字只有前台程序的文件名，没有窗口标题。
type Machine struct {
	PowerKnown bool
	Charging   bool
	Percent    int
	Idle       time.Duration
	IdleKnown  bool
	Focus      string
}

func machine() Machine {
	return readMachine()
}
