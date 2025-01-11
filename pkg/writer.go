package salpidae

import "os"

func WriteFile(fileName string, signature []string) error {
	f, e := os.Create(fileName)
	if e != nil {
		return e
	}
	defer f.Close()

	for _, hash := range signature {
		_, e = f.WriteString(hash + "\n")
		if e != nil {
			return e
		}
	}
	return nil
}
