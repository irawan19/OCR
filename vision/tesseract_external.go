package vision

import (
	"bytes"
	"os/exec"
)

func ExtractTextWithPython(imagePath string) (string, error) {
	cmd := exec.Command("python", "extractor.py", imagePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
